package rest

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/security"
)

type Key int

const (
	RequestLoggerKey Key = iota
	RequestIDKey
)

type loggingResponseWriter struct {
	w      http.ResponseWriter
	buf    bytes.Buffer
	header int
}

func (w *loggingResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	(&w.buf).Write(b)
	return w.w.Write(b)
}

func (w *loggingResponseWriter) WriteHeader(h int) {
	w.header = h
	w.w.WriteHeader(h)
}

func (w *loggingResponseWriter) Content() string {
	return w.buf.String()
}

func RequestLoggerFilter(logger *slog.Logger) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		rq := req.Request

		// search a better way for a unique callid
		// perhaps a reverseproxy in front generates a unique header for some sort
		// of opentracing support?

		requestID := req.HeaderParameter("X-Request-Id")
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				_, _ = resp.Write([]byte("unable to generate request uuid " + err.Error()))
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			requestID = uuid.String()
		}

		fields := []any{
			"rqid", requestID,
			"remoteaddr", rq.RemoteAddr,
			"method", rq.Method,
			"uri", rq.URL.RequestURI(),
			"route", req.SelectedRoutePath(),
		}

		debug := isDebug(logger)

		if debug {
			body, _ := httputil.DumpRequest(rq, true)
			fields = append(fields, "body", string(body))
		}

		// this creates a child log with the given fields as a structured context
		requestLogger := logger.With(fields...)

		enrichedContext := context.WithValue(req.Request.Context(), RequestLoggerKey, requestLogger)
		enrichedContext = context.WithValue(enrichedContext, RequestIDKey, requestID)
		req.Request = req.Request.WithContext(enrichedContext)

		t := time.Now()

		writer := &loggingResponseWriter{w: resp.ResponseWriter}
		resp.ResponseWriter = writer

		chain.ProcessFilter(req, resp)

		afterChainFields := []any{"status", resp.StatusCode(), "content-length", resp.ContentLength(), "duration", time.Since(t).String()}

		// refetch logger. the stack of filters could contain the "UserAuth" filter from below which
		// changes the logger
		requestLogger = GetLoggerFromContext(req.Request, requestLogger)

		if debug || resp.StatusCode() >= 400 {
			afterChainFields = append(afterChainFields, "response", writer.Content())
		}

		if resp.StatusCode() < 400 {
			requestLogger.Info("finished handling rest call", afterChainFields...)
		} else {
			requestLogger.Error("finished handling rest call", afterChainFields...)
		}
	}
}

func UserAuth(ug security.UserGetter, fallbackLogger *slog.Logger) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		log := GetLoggerFromContext(req.Request, fallbackLogger)

		usr, err := ug.User(req.Request)
		if err != nil {
			var hmerr *security.WrongHMAC
			if errors.As(err, &hmerr) {
				log.Error("cannot get user from request", "error", err, "got", hmerr.Got, "want", hmerr.Want)
			} else {
				log.Error("cannot get user from request", "error", err)
			}

			err = resp.WriteHeaderAndEntity(http.StatusForbidden, httperrors.NewHTTPError(http.StatusForbidden, err))
			if err != nil {
				log.Error("error sending response", "error", err)
			}
			return
		}

		rq := req.Request
		ctx := security.PutUserInContext(rq.Context(), usr)

		log = log.With("useremail", usr.EMail, "username", usr.Name, "usertenant", usr.Tenant)
		ctx = context.WithValue(ctx, RequestLoggerKey, log)

		req.Request = req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)
	}
}

func isDebug(log *slog.Logger) bool {
	return log.Enabled(context.Background(), slog.LevelDebug)
}

func GetLoggerFromContext(rq *http.Request, fallback *slog.Logger) *slog.Logger {
	l, ok := rq.Context().Value(RequestLoggerKey).(*slog.Logger)
	if ok {
		return l
	}
	return fallback
}
