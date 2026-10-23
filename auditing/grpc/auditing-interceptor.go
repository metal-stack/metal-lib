package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/metal-stack/metal-lib/auditing"
	"google.golang.org/grpc"
)

type Key int

const (
	// Deprecated: use auditing.Session.RequestID instead.
	RequestIDKey Key = iota
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that audits
// invocations through the given backend. If indexing the request phase fails,
// the invocation is failed; failures indexing the response phase are logged.
func UnaryServerInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.UnaryServerInterceptor, error) {
	rec, err := newRecorder(a, logger)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !shouldAudit(info.FullMethod) {
			return handler(ctx, req)
		}

		sess, err := rec.Begin(ctx, auditing.Transport{
			Kind:      auditing.EntryTypeGRPC,
			Detail:    auditing.EntryDetailGRPCUnary,
			Path:      info.FullMethod,
			RequestID: existingRequestID(ctx),
		})
		if err != nil {
			return nil, err
		}

		childCtx := context.WithValue(sess.Context(), RequestIDKey, sess.RequestID())
		resp, err = handler(childCtx, req)

		if ferr := sess.Finish(auditing.Outcome{Err: err}); ferr != nil {
			logger.Error("unable to index", "error", ferr)
		}
		return resp, err
	}, nil
}

// StreamServerInterceptor returns a gRPC stream server interceptor that audits
// invocations through the given backend. If indexing the opened phase fails,
// the invocation is failed; failures indexing the closed phase are logged.
func StreamServerInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (grpc.StreamServerInterceptor, error) {
	rec, err := newRecorder(a, logger)
	if err != nil {
		return nil, err
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !shouldAudit(info.FullMethod) {
			return handler(srv, ss)
		}

		sess, err := rec.Begin(ss.Context(), auditing.Transport{
			Kind:      auditing.EntryTypeGRPC,
			Detail:    auditing.EntryDetailGRPCStream,
			Path:      info.FullMethod,
			RequestID: existingRequestID(ss.Context()),
		})
		if err != nil {
			return err
		}

		childSS := grpcServerStreamWithContext{
			ServerStream: ss,
			ctx:          context.WithValue(sess.Context(), RequestIDKey, sess.RequestID()),
		}

		err = handler(srv, childSS)
		if ferr := sess.Finish(auditing.Outcome{Err: err}); ferr != nil {
			logger.Error("unable to index", "error", ferr)
		}
		return err
	}, nil
}

func newRecorder(a auditing.Auditing, logger *slog.Logger) (*auditing.Recorder, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create interceptor")
	}
	return auditing.NewRecorder(a, logger)
}

func existingRequestID(ctx context.Context) string {
	if str, ok := ctx.Value(RequestIDKey).(string); ok {
		return str
	}
	return ""
}

// GrpcError is the serialized shape of a gRPC status error in audit entries.
//
// Deprecated: produced by auditing.SerializableError; kept for compatibility.
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
