package auditing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Include explicitly includes the request to the auditing backend even if the request method would prevent the request to be audited (only applies for the http filter)
	Include string = "include-to-auditing"
	// Exclude explicitly excludes the request to the auditing backend even if the request method would audit the request (only applies for the http filter)
	Exclude string = "exclude-from-auditing"
)

func UnaryServerInterceptor(a Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.UnaryServerInterceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create unary server interceptor")
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if !shouldAudit(info.FullMethod) {
			return handler(ctx, req)
		}
		var requestID string
		if str, ok := ctx.Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			requestID = uuid.String()
		}

		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Type:      EntryTypeGRPC,
			Detail:    EntryDetailGRPCUnary,
			Path:      info.FullMethod,
			Phase:     EntryPhaseRequest,
		}

		user := security.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err = a.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.prepareForNextPhase()
		resp, err = handler(childCtx, req)

		auditReqContext.Phase = EntryPhaseResponse
		auditReqContext.Body = resp
		auditReqContext.StatusCode = statusCodeFromGrpc(err)

		if err != nil {
			auditReqContext.Error = err
			err2 := a.Index(auditReqContext)
			if err2 != nil {
				logger.Error("unable to index", "error", err2)
			}
			return nil, err
		}

		err = a.Index(auditReqContext)
		return resp, err
	}, nil
}

func StreamServerInterceptor(a Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.StreamServerInterceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create stream server interceptor")
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !shouldAudit(info.FullMethod) {
			return handler(srv, ss)
		}
		var requestID string
		if str, ok := ss.Context().Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ss.Context(), rest.RequestIDKey, requestID)
		childSS := grpcServerStreamWithContext{
			ServerStream: ss,
			ctx:          childCtx,
		}

		auditReqContext := Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    EntryDetailGRPCStream,
			Path:      info.FullMethod,
			Phase:     EntryPhaseOpened,
			Type:      EntryTypeGRPC,
		}

		user := security.GetUserFromContext(ss.Context())
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.Index(auditReqContext)
		if err != nil {
			return err
		}

		auditReqContext.prepareForNextPhase()
		err = handler(srv, childSS)
		auditReqContext.StatusCode = statusCodeFromGrpc(err)

		if err != nil {
			auditReqContext.Error = err
			err2 := a.Index(auditReqContext)
			if err2 != nil {
				logger.Error("unable to index", "error", err2)
			}
			return err
		}

		auditReqContext.Phase = EntryPhaseClosed
		err = a.Index(auditReqContext)

		return err
	}, nil
}

type auditingConnectInterceptor struct {
	auditing    Auditing
	logger      *slog.Logger
	shouldAudit func(fullMethod string) bool
}

// WrapStreamingClient implements connect.Interceptor
func (a auditingConnectInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		if !a.shouldAudit(s.Procedure) {
			return next(ctx, s)
		}
		var requestID string
		if str, ok := ctx.Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				a.logger.Error("unable to generate uuid", "error", err)
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    EntryDetailGRPCStream,
			Path:      s.Procedure,
			Phase:     EntryPhaseOpened,
			Type:      EntryTypeGRPC,
		}

		user := security.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.prepareForNextPhase()
		scc := next(childCtx, s)

		auditReqContext.Phase = EntryPhaseClosed
		auditReqContext.StatusCode = statusCodeFromGrpc(err)

		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		return scc
	}
}

// WrapStreamingHandler implements connect.Interceptor
func (a auditingConnectInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, shc connect.StreamingHandlerConn) error {
		if !a.shouldAudit(shc.Spec().Procedure) {
			return next(ctx, shc)
		}
		var requestID string
		if str, ok := ctx.Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       EntryDetailGRPCStream,
			Path:         shc.Spec().Procedure,
			Phase:        EntryPhaseOpened,
			Type:         EntryTypeGRPC,
			RemoteAddr:   shc.RequestHeader().Get("X-Real-Ip"),
			ForwardedFor: shc.RequestHeader().Get("X-Forwarded-For"),
		}

		if auditReqContext.RemoteAddr == "" {
			auditReqContext.RemoteAddr = shc.Peer().Addr
		}

		user := security.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.prepareForNextPhase()
		err = next(childCtx, shc)
		auditReqContext.StatusCode = statusCodeFromGrpc(err)

		if err != nil {
			auditReqContext.Error = err
			err2 := a.auditing.Index(auditReqContext)
			if err2 != nil {
				a.logger.Error("unable to index", "error", err2)
			}
			return err
		}

		auditReqContext.Phase = EntryPhaseClosed
		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		return err
	}
}

// WrapUnary implements connect.Interceptor
func (i auditingConnectInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		if !i.shouldAudit(ar.Spec().Procedure) {
			return next(ctx, ar)
		}
		var requestID string
		if str, ok := ctx.Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       EntryDetailGRPCUnary,
			Path:         ar.Spec().Procedure,
			Phase:        EntryPhaseRequest,
			Type:         EntryTypeGRPC,
			Body:         ar.Any(),
			RemoteAddr:   ar.Header().Get("X-Real-Ip"),
			ForwardedFor: ar.Header().Get("X-Forwarded-For"),
		}

		if auditReqContext.RemoteAddr == "" {
			auditReqContext.RemoteAddr = ar.Peer().Addr
		}

		user := security.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}
		err := i.auditing.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.prepareForNextPhase()

		resp, err := next(childCtx, ar)

		auditReqContext.Phase = EntryPhaseResponse
		if resp != nil {
			auditReqContext.Body = resp.Any()
		}
		auditReqContext.StatusCode = statusCodeFromGrpc(err)

		if err != nil {
			auditReqContext.Error = err
			err2 := i.auditing.Index(auditReqContext)
			if err2 != nil {
				i.logger.Error("unable to index", "error", err2)
			}
			return nil, err
		}

		err = i.auditing.Index(auditReqContext)
		return resp, err
	}
}

func NewConnectInterceptor(a Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connect.Interceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create connect interceptor")
	}
	return auditingConnectInterceptor{
		auditing:    a,
		logger:      logger,
		shouldAudit: shouldAudit,
	}, nil
}

type (
	httpFilterOpt interface{}

	httpFilterErrorCallback struct {
		callback func(err error, response *restful.Response)
	}
)

func NewHttpFilterErrorCallback(callback func(err error, response *restful.Response)) *httpFilterErrorCallback {
	return &httpFilterErrorCallback{callback: callback}
}

func HttpFilter(a Auditing, logger *slog.Logger, opts ...httpFilterOpt) (restful.FilterFunction, error) {
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
		auditReqContext := Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Type:         EntryTypeHTTP,
			Detail:       EntryDetail(r.Method),
			Path:         r.URL.Path,
			Phase:        EntryPhaseRequest,
			ForwardedFor: request.HeaderParameter("x-forwarded-for"),
			RemoteAddr:   r.RemoteAddr,
		}
		user := security.GetUserFromContext(r.Context())
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

		auditReqContext.prepareForNextPhase()
		chain.ProcessFilter(request, response)

		auditReqContext.Phase = EntryPhaseResponse
		auditReqContext.StatusCode = pointer.Pointer(response.StatusCode())
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

type grpcServerStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

// Context implements grpc.ServerStream
func (s grpcServerStreamWithContext) Context() context.Context {
	return s.ctx
}

func statusCodeFromGrpc(err error) *int {
	s, ok := status.FromError(err)
	if !ok {
		return pointer.Pointer(int(codes.Unknown))
	}

	return pointer.Pointer(int(s.Code()))
}
