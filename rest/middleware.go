package rest

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
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

func RequestLogger(debug bool, logger *zap.Logger) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		rq := req.Request
		// search a better way for a unique callid
		// perhaps a reverseproxy in front generates a unique header for som sort
		// of opentracing support?

		requestID := req.HeaderParameter("X-Request-Id")
		rqid := zap.String("rqid", requestID)
		if requestID == "" {
			ts := time.Now().UnixNano()
			rqid = zap.Int64("rqid", ts)
		}
		fields := []zap.Field{
			rqid,
			zap.String("remoteaddr", strings.Split(rq.RemoteAddr, ":")[0]),
			zap.String("method", rq.Method),
			zap.String("uri", rq.URL.RequestURI()),
			zap.String("route", req.SelectedRoutePath()),
		}

		if debug {
			body, _ := httputil.DumpRequest(rq, true)
			fields = append(fields, zap.String("body", string(body)))
			resp.ResponseWriter = &loggingResponseWriter{w: resp.ResponseWriter}
		}

		rqlogger := logger.With(fields...)
		rq = req.Request.WithContext(zapup.PutLogger(req.Request.Context(), rqlogger))
		req.Request = rq
		t := time.Now()

		chain.ProcessFilter(req, resp)
		fields = append(fields, zap.Int("status", resp.StatusCode()), zap.Int("content-length", resp.ContentLength()), zap.Duration("duration", time.Since(t)))

		// refetch logger. the stack of filters could contain the "UserAuth" filter from below which
		// changes the logger

		innerlogger := zapup.RequestLogger(req.Request)

		if debug {
			fields = append(fields, zap.String("response", resp.ResponseWriter.(*loggingResponseWriter).Content()))
		}
		if resp.StatusCode() < 400 {
			innerlogger.Info("Rest Call", fields...)
		} else {
			innerlogger.Error("Rest Call", fields...)
		}
	}
}

func UserAuth(ug security.UserGetter) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		log := zapup.RequestLogger(req.Request)
		usr, err := ug.User(req.Request)
		if err != nil {
			if hmerr, ok := err.(*security.WrongHMAC); ok {
				log.Error("cannot get user from request", zap.Error(err), zap.String("got", hmerr.Got), zap.String("want", hmerr.Want))
			} else {
				log.Error("cannot get user from request", zap.Error(err))
			}
			err = resp.WriteHeaderAndEntity(http.StatusForbidden, httperrors.NewHTTPError(http.StatusForbidden, err))
			if err != nil {
				log.Error("writeHeaderAndEntity", zap.Error(err))
			}
			return
		}
		log = log.With(zap.String("useremail", usr.EMail))
		rq := req.Request
		ctx := security.PutUserInContext(zapup.PutLogger(rq.Context(), log), usr)
		req.Request = rq.WithContext(ctx)
		chain.ProcessFilter(req, resp)
	}
}
