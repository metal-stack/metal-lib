package auditing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	// Include explicitly includes the request to the auditing backend even if the request method would prevent the request to be audited (only applies for the http filter)
	Include string = "include-to-auditing"
	// Exclude explicitly excludes the request to the auditing backend even if the request method would audit the request (only applies for the http filter)
	Exclude string = "exclude-from-auditing"
)

func UnaryServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.UnaryServerInterceptor {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create unary server interceptor")
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
			requestID = uuid.NewString()
		}

		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
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
		}

		err = a.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.prepareForNextPhase()
		resp, err = handler(childCtx, req)
		auditReqContext.Phase = EntryPhaseResponse
		auditReqContext.Body = resp
		if err != nil {
			auditReqContext.Error = err
			err2 := a.Index(auditReqContext)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return nil, err
		}

		err = a.Index(auditReqContext)
		return resp, err
	}
}

func StreamServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.StreamServerInterceptor {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create stream server interceptor")
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
			requestID = uuid.NewString()
		}
		childCtx := context.WithValue(ss.Context(), rest.RequestIDKey, requestID)
		childSS := grpcServerStreamWithContext{
			ServerStream: ss,
			ctx:          childCtx,
		}

		auditReqContext := Entry{
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
		}

		err := a.Index(auditReqContext)
		if err != nil {
			return err
		}

		auditReqContext.prepareForNextPhase()
		err = handler(srv, childSS)
		if err != nil {
			auditReqContext.Error = err
			err2 := a.Index(auditReqContext)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return err
		}

		auditReqContext.Phase = EntryPhaseClosed
		err = a.Index(auditReqContext)
		return err
	}
}

type auditingConnectInterceptor struct {
	auditing    Auditing
	logger      *zap.SugaredLogger
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
			requestID = uuid.NewString()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
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
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Errorf("unable to index error: %v", err)
		}

		auditReqContext.prepareForNextPhase()
		scc := next(childCtx, s)
		auditReqContext.Phase = EntryPhaseClosed

		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Errorf("unable to index error: %v", err)
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
			requestID = uuid.NewString()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
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
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Errorf("unable to index error: %v", err)
		}

		auditReqContext.prepareForNextPhase()
		err = next(childCtx, shc)
		if err != nil {
			auditReqContext.Error = err
			err2 := a.auditing.Index(auditReqContext)
			if err2 != nil {
				a.logger.Errorf("unable to index error: %v", err2)
			}
			return err
		}

		auditReqContext.Phase = EntryPhaseClosed
		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Errorf("unable to index error: %v", err)
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
			requestID = uuid.NewString()
		}
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := Entry{
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
		}
		err := i.auditing.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.prepareForNextPhase()
		resp, err := next(childCtx, ar)
		auditReqContext.Phase = EntryPhaseResponse
		auditReqContext.Body = resp
		if err != nil {
			auditReqContext.Error = err
			err2 := i.auditing.Index(auditReqContext)
			if err2 != nil {
				i.logger.Errorf("unable to index error: %v", err2)
			}
			return nil, err
		}
		err = i.auditing.Index(auditReqContext)
		return resp, err
	}
}

func NewConnectInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) connect.Interceptor {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create connect interceptor")
	}
	return auditingConnectInterceptor{
		auditing:    a,
		logger:      logger,
		shouldAudit: shouldAudit,
	}
}

func HttpFilter(a Auditing, logger *zap.SugaredLogger) restful.FilterFunction {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create http middleware")
	}
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		r := request.Request

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			break
		default:
			if request.SelectedRoute() == nil {
				logger.Debugw("selected route is not defined, continue request processing")
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
			logger.Debugw("selected route is not defined, continue request processing")
			chain.ProcessFilter(request, response)
			return
		}

		excluded, ok := request.SelectedRoute().Metadata()[Exclude].(bool)
		if ok && excluded {
			logger.Debugw("excluded route from auditing through metadata annotation", "path", request.SelectedRoute().Path())
			chain.ProcessFilter(request, response)
			return
		}

		var requestID string
		if str, ok := r.Context().Value(rest.RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			requestID = uuid.NewString()
		}
		auditReqContext := Entry{
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
		}

		if r.Method != http.MethodGet && r.Body != nil {
			bodyReader := r.Body
			body, err := io.ReadAll(bodyReader)
			r.Body = io.NopCloser(bytes.NewReader(body))
			if err != nil {
				logger.Errorf("unable to read request body: %v", err)
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
			err = json.Unmarshal(body, &auditReqContext.Body)
			if err != nil {
				auditReqContext.Body = string(body)
			}
		}

		err := a.Index(auditReqContext)
		if err != nil {
			logger.Errorf("unable to index error: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}

		bufferedResponseWriter := &bufferedHttpResponseWriter{
			w: response.ResponseWriter,
		}
		response.ResponseWriter = bufferedResponseWriter

		auditReqContext.prepareForNextPhase()
		chain.ProcessFilter(request, response)

		auditReqContext.Phase = EntryPhaseResponse
		auditReqContext.StatusCode = response.StatusCode()
		strBody := bufferedResponseWriter.Content()
		body := []byte(strBody)
		err = json.Unmarshal(body, &auditReqContext.Body)
		if err != nil {
			auditReqContext.Body = strBody
			auditReqContext.Error = err
		}

		err = a.Index(auditReqContext)
		if err != nil {
			logger.Errorf("unable to index error: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
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
