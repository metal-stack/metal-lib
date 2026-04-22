package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/auditing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Key int

const (
	RequestIDKey Key = iota
)

func UnaryServerInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.UnaryServerInterceptor, error) {
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

		auditReqContext := auditing.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Type:      auditing.EntryTypeGRPC,
			Detail:    auditing.EntryDetailGRPCUnary,
			Path:      info.FullMethod,
			Phase:     auditing.EntryPhaseRequest,
		}

		user := auditing.GetUserFromContext(ctx)
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

		auditReqContext.Phase = auditing.EntryPhaseResponse
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

func StreamServerInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.StreamServerInterceptor, error) {
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

		auditReqContext := auditing.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    auditing.EntryDetailGRPCStream,
			Path:      info.FullMethod,
			Phase:     auditing.EntryPhaseOpened,
			Type:      auditing.EntryTypeGRPC,
		}

		user := auditing.GetUserFromContext(ss.Context())
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

		auditReqContext.Phase = auditing.EntryPhaseClosed
		err = a.Index(auditReqContext)

		return err
	}, nil
}

func statusCodeFromGrpcOrConnect(err error) *int {
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
