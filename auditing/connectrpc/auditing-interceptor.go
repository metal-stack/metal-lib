package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	connectv2 "connectrpc.com/connect/v2"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/auditing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Key int

const (
	RequestIDKey Key = iota
)

type auditingConnectInterceptor struct {
	auditing    auditing.Auditing
	logger      *slog.Logger
	shouldAudit func(fullMethod string) bool
}

func NewConnectInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connect.Interceptor, error) {
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
		requestID, err := requestIDFromContext(ctx)
		if err != nil {
			a.logger.Error("unable to generate uuid", "error", err)
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := a.baseEntry(ctx, requestID, s.Procedure)
		auditReqContext.Detail = auditing.EntryDetailGRPCStream
		auditReqContext.Phase = auditing.EntryPhaseOpened

		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.PrepareForNextPhase()
		scc := next(childCtx, s)

		auditReqContext.Phase = auditing.EntryPhaseClosed
		auditReqContext.StatusCode = statusCode(err)

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
		requestID, err := requestIDFromContext(ctx)
		if err != nil {
			return err
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := a.baseEntry(ctx, requestID, shc.Spec().Procedure)
		auditReqContext.Detail = auditing.EntryDetailGRPCStream
		auditReqContext.Phase = auditing.EntryPhaseOpened
		setRemoteAddr(&auditReqContext, shc.RequestHeader().Get, shc.Peer().Addr)

		err = a.auditing.Index(auditReqContext)
		if err != nil {
			a.logger.Error("unable to index", "error", err)
		}

		auditReqContext.PrepareForNextPhase()
		err = next(childCtx, shc)
		auditReqContext.StatusCode = statusCode(err)

		if err != nil {
			auditReqContext.Error = serializableError(err)

			err2 := a.auditing.Index(auditReqContext)
			if err2 != nil {
				a.logger.Error("unable to index", "error", err2)
			}
			return err
		}

		auditReqContext.Phase = auditing.EntryPhaseClosed
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
		requestID, err := requestIDFromContext(ctx)
		if err != nil {
			return nil, err
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		auditReqContext := i.baseEntry(ctx, requestID, ar.Spec().Procedure)
		auditReqContext.Detail = auditing.EntryDetailGRPCUnary
		auditReqContext.Phase = auditing.EntryPhaseRequest
		auditReqContext.Body = ar.Any()
		setRemoteAddr(&auditReqContext, ar.Header().Get, ar.Peer().Addr)

		err = i.auditing.Index(auditReqContext)
		if err != nil {
			return nil, err
		}

		auditReqContext.PrepareForNextPhase()

		resp, err := next(childCtx, ar)

		auditReqContext.Phase = auditing.EntryPhaseResponse
		auditReqContext.StatusCode = statusCode(err)

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

// requestIDFromContext returns the request id stored in ctx, or generates a new
// one if none is present. It is shared between the connect-go v1 and v2
// interceptors.
func requestIDFromContext(ctx context.Context) (string, error) {
	if str, ok := ctx.Value(RequestIDKey).(string); ok && str != "" {
		return str, nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// baseEntry fills the audit entry fields shared by all interceptor variants.
func (a auditingConnectInterceptor) baseEntry(ctx context.Context, requestID, procedure string) auditing.Entry {
	entry := auditing.Entry{
		Timestamp: time.Now(),
		RequestId: requestID,
		Path:      procedure,
		Type:      auditing.EntryTypeGRPC,
	}

	user := auditing.GetUserFromContext(ctx)
	if user != nil {
		entry.User = user.Subject
		entry.Tenant = user.Tenant
		entry.Project = user.Project
	}

	return entry
}

// setRemoteAddr fills the remote address fields from the given header getter,
// falling back to the peer address.
func setRemoteAddr(entry *auditing.Entry, headerGet func(string) string, peerAddr string) {
	entry.RemoteAddr = headerGet("X-Real-Ip")
	entry.ForwardedFor = headerGet("X-Forwarded-For")
	if entry.RemoteAddr == "" {
		entry.RemoteAddr = peerAddr
	}
}

func statusCode(err error) *int {
	if err == nil {
		return new(int)
	}

	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		return new(int(connectErr.Code()))
	}

	if connectErrV2, ok := errors.AsType[*connectv2.Error](err); ok {
		return new(int(connectErrV2.Code()))
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

	if connectErrV2, ok := errors.AsType[*connectv2.Error](err); ok {
		return ConnectError{
			Code:    uint32(connectErrV2.Code()),
			Message: connectErrV2.Code().String(),
			Err:     connectErrV2.Error(),
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
