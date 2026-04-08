package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/auditing/api"
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

func HttpFilter(a api.Auditing, logger *slog.Logger, opts ...httpFilterOpt) (restful.FilterFunction, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create http middleware")
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

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			break
		default:
			if request.SelectedRoute() == nil {
				logger.Debug("selected route is not defined, continue request processing")
				chain.ProcessFilter(request, response)
				return
			}
			included, ok := request.SelectedRoute().Metadata()[Include].(bool)
			if ok && included {
				break
			} else {
				chain.ProcessFilter(request, response)
				return
			}
		}

		if request.SelectedRoute() == nil {
			logger.Debug("selected route is not defined, continue request processing")
			chain.ProcessFilter(request, response)
			return
		}

		excluded, ok := request.SelectedRoute().Metadata()[Exclude].(bool)
		if ok && excluded {
			logger.Debug("excluded route from auditing through metadata annotation", "path", request.SelectedRoute().Path())
			chain.ProcessFilter(request, response)
			return
		}

		var requestID string
		if str, ok := r.Context().Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				logger.Error("unable to generate uuid", "error", err)
				_, _ = response.Write([]byte("unable to generate request uuid " + err.Error()))
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
			requestID = uuid.String()
		}
		auditReqContext := api.Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Type:         api.EntryTypeHTTP,
			Detail:       api.EntryDetail(r.Method),
			Path:         r.URL.Path,
			Phase:        api.EntryPhaseRequest,
			ForwardedFor: request.HeaderParameter("x-forwarded-for"),
			RemoteAddr:   r.RemoteAddr,
		}
		user := api.GetUserFromContext(r.Context())
		if user != nil {
			auditReqContext.User = user.EMail
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		if r.Method != http.MethodGet && r.Body != nil {
			bodyReader := r.Body
			body, err := io.ReadAll(bodyReader)
			r.Body = io.NopCloser(bytes.NewReader(body))
			if err != nil {
				logger.Error("unable to read request body", "error", err)
				errorCallback(err, response)
				return
			}
			err = json.Unmarshal(body, &auditReqContext.Body)
			if err != nil {
				auditReqContext.Body = string(body)
			}
		}

		err := a.Index(auditReqContext)
		if err != nil {
			logger.Error("unable to index", "error", err)
			errorCallback(err, response)
			return
		}

		bufferedResponseWriter := &bufferedHttpResponseWriter{
			w: response.ResponseWriter,
		}
		response.ResponseWriter = bufferedResponseWriter

		auditReqContext.PrepareForNextPhase()
		chain.ProcessFilter(request, response)

		auditReqContext.Phase = api.EntryPhaseResponse
		auditReqContext.StatusCode = new(response.StatusCode())
		strBody := bufferedResponseWriter.Content()
		body := []byte(strBody)
		err = json.Unmarshal(body, &auditReqContext.Body)
		if err != nil {
			auditReqContext.Body = strBody
			auditReqContext.Error = err
		}

		err = a.Index(auditReqContext)
		if err != nil {
			logger.Error("unable to index", "error", err)
			errorCallback(err, response)
			return
		}
	}, nil
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
