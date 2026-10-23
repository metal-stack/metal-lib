package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/rest"
)

const (
	// Include explicitly includes the request to the auditing backend even if the request method would prevent the request to be audited (only applies for the http filter)
	Include string = "include-to-auditing"
	// Exclude explicitly excludes the request to the auditing backend even if the request method would audit the request (only applies for the http filter)
	Exclude string = "exclude-from-auditing"
)

type (
	httpFilterOpt any

	httpFilterErrorCallback struct {
		callback func(err error, response *restful.Response)
	}
)

func NewHttpFilterErrorCallback(callback func(err error, response *restful.Response)) *httpFilterErrorCallback {
	return &httpFilterErrorCallback{callback: callback}
}

// HttpFilter returns a restful filter that audits requests through the given
// backend. If indexing the request phase fails, the request is failed via the
// error callback; failures indexing the response phase are logged.
func HttpFilter(a auditing.Auditing, logger *slog.Logger, opts ...httpFilterOpt) (restful.FilterFunction, error) {
	rec, err := auditing.NewRecorder(a, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot use nil auditing to create http middleware: %w", err)
	}

	errorCallback := func(err error, response *restful.Response) {
		if err := response.WriteError(http.StatusInternalServerError, err); err != nil {
			logger.Error("unable to write http response", "error", err)
		}
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case *httpFilterErrorCallback:
			errorCallback = o.callback
		default:
			return nil, fmt.Errorf("unknown filter option: %T", opt)
		}
	}

	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		r := request.Request

		if !shouldAuditRoute(logger, request, r.Method) {
			chain.ProcessFilter(request, response)
			return
		}

		var requestBody any
		if r.Method != http.MethodGet && r.Body != nil {
			raw, err := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(raw))
			if err != nil {
				logger.Error("unable to read request body", "error", err)
				errorCallback(err, response)
				return
			}
			if err := json.Unmarshal(raw, &requestBody); err != nil {
				requestBody = string(raw)
			}
		}

		sess, err := rec.Begin(r.Context(), auditing.Transport{
			Kind:         auditing.EntryTypeHTTP,
			Detail:       auditing.EntryDetail(r.Method),
			Path:         r.URL.Path,
			ForwardedFor: request.HeaderParameter("x-forwarded-for"),
			RemoteAddr:   r.RemoteAddr,
			RequestID:    existingRequestID(r),
			Body:         requestBody,
		})
		if err != nil {
			logger.Error("unable to index", "error", err)
			errorCallback(err, response)
			return
		}

		bufferedResponseWriter := &bufferedHttpResponseWriter{
			w: response.ResponseWriter,
		}
		response.ResponseWriter = bufferedResponseWriter

		chain.ProcessFilter(request, response)

		strBody := bufferedResponseWriter.Content()
		var responseBody any
		if err := json.Unmarshal([]byte(strBody), &responseBody); err != nil {
			responseBody = strBody
		}

		if ferr := sess.Finish(auditing.Outcome{
			StatusCode: response.StatusCode(),
			Body:       responseBody,
		}); ferr != nil {
			logger.Error("unable to index", "error", ferr)
		}
	}, nil
}

// shouldAuditRoute applies the transport-specific routing policy: mutating
// methods are always audited, others only when the route opts in via the
// Include metadata annotation. Routes opting out via Exclude are never
// audited.
func shouldAuditRoute(logger *slog.Logger, request *restful.Request, method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		break
	default:
		if request.SelectedRoute() == nil {
			logger.Debug("selected route is not defined, continue request processing")
			return false
		}
		included, ok := request.SelectedRoute().Metadata()[Include].(bool)
		if !ok || !included {
			return false
		}
	}

	if request.SelectedRoute() == nil {
		logger.Debug("selected route is not defined, continue request processing")
		return false
	}

	excluded, ok := request.SelectedRoute().Metadata()[Exclude].(bool)
	if ok && excluded {
		logger.Debug("excluded route from auditing through metadata annotation", "path", request.SelectedRoute().Path())
		return false
	}

	return true
}

func existingRequestID(r *http.Request) string {
	if str, ok := r.Context().Value(rest.RequestIDKey).(string); ok {
		return str
	}
	return ""
}

type bufferedHttpResponseWriter struct {
	w http.ResponseWriter

	buf    bytes.Buffer
	header int
}

func (w *bufferedHttpResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *bufferedHttpResponseWriter) Write(b []byte) (int, error) {
	(&w.buf).Write(b)
	return w.w.Write(b)
}

func (w *bufferedHttpResponseWriter) WriteHeader(h int) {
	w.header = h
	w.w.WriteHeader(h)
}

func (w *bufferedHttpResponseWriter) Content() string {
	return w.buf.String()
}
