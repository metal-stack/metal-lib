package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/auditing/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Key int

const (
	RequestLoggerKey Key = iota
	RequestIDKey
)

type auditingConnectInterceptor struct {
	auditing    api.Auditing
	logger      *slog.Logger
	shouldAudit func(fullMethod string) bool
}

func NewConnectInterceptor(a api.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connect.Interceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create connect interceptor")
	}
	return auditingConnectInterceptor{
		auditing:    a,
		logger:      logger,
		shouldAudit: shouldAudit,
	}, nil
}

// WrapStreamingClient implements connect.Interceptor
func (a auditingConnectInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		if !a.shouldAudit(s.Procedure) {
			return next(ctx, s)
		}
		var requestID string
		if str, ok := ctx.Value(RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				a.logger.Error("unable to generate uuid", "error", err)
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := api.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    api.EntryDetailGRPCStream,
			Path:      s.Procedure,
			Phase:     api.EntryPhaseOpened,
			Type:      api.EntryTypeGRPC,
		}

		user := api.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.PrepareForNextPhase()
		scc := next(childCtx, s)

		auditReqContext.Phase = api.EntryPhaseClosed
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

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
		if str, ok := ctx.Value(RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := api.Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       api.EntryDetailGRPCStream,
			Path:         shc.Spec().Procedure,
			Phase:        api.EntryPhaseOpened,
			Type:         api.EntryTypeGRPC,
			RemoteAddr:   shc.RequestHeader().Get("X-Real-Ip"),
			ForwardedFor: shc.RequestHeader().Get("X-Forwarded-For"),
		}

		if auditReqContext.RemoteAddr == "" {
			auditReqContext.RemoteAddr = shc.Peer().Addr
		}

		user := api.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.PrepareForNextPhase()
		err = next(childCtx, shc)
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = serializableError(err)

			err2 := a.auditing.Index(auditReqContext)
			if err2 != nil {
				a.logger.Error("unable to index", "error", err2)
			}
			return err
		}

		auditReqContext.Phase = api.EntryPhaseClosed
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
		if str, ok := ctx.Value(RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := api.Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       api.EntryDetailGRPCUnary,
			Path:         ar.Spec().Procedure,
			Phase:        api.EntryPhaseRequest,
			Type:         api.EntryTypeGRPC,
			Body:         ar.Any(),
			RemoteAddr:   ar.Header().Get("X-Real-Ip"),
			ForwardedFor: ar.Header().Get("X-Forwarded-For"),
		}

		if auditReqContext.RemoteAddr == "" {
			auditReqContext.RemoteAddr = ar.Peer().Addr
		}

		user := api.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}
		err := i.auditing.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.PrepareForNextPhase()

		resp, err := next(childCtx, ar)

		auditReqContext.Phase = api.EntryPhaseResponse
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = serializableError(err)

			err2 := i.auditing.Index(auditReqContext)
			if err2 != nil {
				i.logger.Error("unable to index", "error", err2)
			}
			return nil, err
		} else if resp != nil {
			auditReqContext.Body = resp.Any()
		}

		err = i.auditing.Index(auditReqContext)
		return resp, err
	}
}

func UnaryServerInterceptor(a api.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.UnaryServerInterceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create unary server interceptor")
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !shouldAudit(info.FullMethod) {
			return handler(ctx, req)
		}
		var requestID string
		if str, ok := ctx.Value(RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			requestID = uuid.String()
		}

		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := api.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Type:      api.EntryTypeGRPC,
			Detail:    api.EntryDetailGRPCUnary,
			Path:      info.FullMethod,
			Phase:     api.EntryPhaseRequest,
		}

		user := api.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err = a.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.PrepareForNextPhase()
		resp, err = handler(childCtx, req)

		auditReqContext.Phase = api.EntryPhaseResponse
		auditReqContext.Body = resp
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = serializableError(err)

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

func StreamServerInterceptor(a api.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.StreamServerInterceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create stream server interceptor")
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !shouldAudit(info.FullMethod) {
			return handler(srv, ss)
		}
		var requestID string
		if str, ok := ss.Context().Value(RequestIDKey).(string); ok {
			requestID = str
		}
		if requestID == "" {
			uuid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			requestID = uuid.String()
		}
		childCtx := context.WithValue(ss.Context(), RequestIDKey, requestID)
		childSS := grpcServerStreamWithContext{
			ServerStream: ss,
			ctx:          childCtx,
		}

		auditReqContext := api.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    api.EntryDetailGRPCStream,
			Path:      info.FullMethod,
			Phase:     api.EntryPhaseOpened,
			Type:      api.EntryTypeGRPC,
		}

		user := api.GetUserFromContext(ss.Context())
		if user != nil {
			auditReqContext.User = user.Subject
			auditReqContext.Tenant = user.Tenant
			auditReqContext.Project = user.Project
		}

		err := a.Index(auditReqContext)
		if err != nil {
			return err
		}

		auditReqContext.PrepareForNextPhase()
		err = handler(srv, childSS)
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = serializableError(err)

			err2 := a.Index(auditReqContext)
			if err2 != nil {
				logger.Error("unable to index", "error", err2)
			}
			return err
		}

		auditReqContext.Phase = api.EntryPhaseClosed
		err = a.Index(auditReqContext)

		return err
	}, nil
}

func statusCodeFromGrpcOrConnect(err error) *int {
	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		return new(int(connectErr.Code()))
	}

	s, ok := status.FromError(err)
	if !ok {
		return new(int(codes.Unknown))
	}

	return new(int(s.Code()))
}

// SerializableError attempts to turn an error into something that is usable for the audit backends.
//
// most errors do not contain public fields (e.g. connect error) and when being serialized will turn into
// an empty map.
//
// some error types (e.g. httperror of this library) can be serialized without any issues, so these
// should stay untouched.
func serializableError(err error) any {
	if err == nil {
		return nil
	}

	var connectErr *connect.Error
	if ok := errors.As(err, &connectErr); ok {
		return ConnectError{
			Code:    uint32(connectErr.Code()),
			Message: connectErr.Code().String(),
			Err:     connectErr.Error(),
		}
	}

	s, ok := status.FromError(err)
	if ok {
		return GrpcError{
			Code:    uint32(s.Code()),
			Message: s.Code().String(),
			Err:     s.Message(),
		}
	}

	// fallback to string (which is better than nothing)
	return struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
}

type ConnectError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Err     string `json:"error"`
}

func (c ConnectError) Error() string {
	return fmt.Sprintf("%s (%d %s)", c.Err, c.Code, c.Message)
}

type GrpcError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Err     string `json:"error"`
}

func (c GrpcError) Error() string {
	return fmt.Sprintf("%s (%d %s)", c.Err, c.Code, c.Message)
}

type grpcServerStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

// Context implements grpc.ServerStream
func (s grpcServerStreamWithContext) Context() context.Context {
	return s.ctx
}
