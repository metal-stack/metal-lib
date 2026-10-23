package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/metal-stack/metal-lib/auditing"
)

type Key int

const (
	// Deprecated: use auditing.Session.RequestID instead.
	RequestIDKey Key = iota
)

type auditingConnectInterceptor struct {
	recorder    *auditing.Recorder
	logger      *slog.Logger
	shouldAudit func(fullMethod string) bool
}

// NewConnectInterceptor returns a connect interceptor that audits invocations
// through the given backend. Indexing failures never fail the invocation;
// they are logged.
func NewConnectInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connect.Interceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create connect interceptor")
	}
	rec, err := auditing.NewRecorder(a, logger)
	if err != nil {
		return nil, err
	}
	return auditingConnectInterceptor{
		recorder:    rec,
		logger:      logger,
		shouldAudit: shouldAudit,
	}, nil
}

// WrapStreamingClient implements connect.Interceptor
func (i auditingConnectInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		if !i.shouldAudit(s.Procedure) {
			return next(ctx, s)
		}

		sess, err := i.recorder.Begin(ctx, auditing.Transport{
			Kind:      auditing.EntryTypeGRPC,
			Detail:    auditing.EntryDetailGRPCStream,
			Path:      s.Procedure,
			RequestID: existingRequestID(ctx),
		})
		if err != nil {
			i.logger.Error("unable to index", "error", err)
			return next(ctx, s)
		}

		childCtx := context.WithValue(sess.Context(), RequestIDKey, sess.RequestID())
		scc := next(childCtx, s)

		// NOTE: the connect interceptor surface offers no hook for stream
		// completion, so the closed phase is recorded when the connection is
		// established.
		if ferr := sess.Finish(auditing.Outcome{}); ferr != nil {
			i.logger.Error("unable to index", "error", ferr)
		}

		return scc
	}
}

// WrapStreamingHandler implements connect.Interceptor
func (i auditingConnectInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, shc connect.StreamingHandlerConn) error {
		if !i.shouldAudit(shc.Spec().Procedure) {
			return next(ctx, shc)
		}

		remoteAddr := shc.RequestHeader().Get("X-Real-Ip")
		if remoteAddr == "" {
			remoteAddr = shc.Peer().Addr
		}

		sess, err := i.recorder.Begin(ctx, auditing.Transport{
			Kind:         auditing.EntryTypeGRPC,
			Detail:       auditing.EntryDetailGRPCStream,
			Path:         shc.Spec().Procedure,
			RemoteAddr:   remoteAddr,
			ForwardedFor: shc.RequestHeader().Get("X-Forwarded-For"),
			RequestID:    existingRequestID(ctx),
		})
		if err != nil {
			i.logger.Error("unable to index", "error", err)
			return next(ctx, shc)
		}

		childCtx := context.WithValue(sess.Context(), RequestIDKey, sess.RequestID())
		err = next(childCtx, shc)
		if ferr := sess.Finish(auditing.Outcome{Err: err}); ferr != nil {
			i.logger.Error("unable to index", "error", ferr)
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

		remoteAddr := ar.Header().Get("X-Real-Ip")
		if remoteAddr == "" {
			remoteAddr = ar.Peer().Addr
		}

		sess, err := i.recorder.Begin(ctx, auditing.Transport{
			Kind:         auditing.EntryTypeGRPC,
			Detail:       auditing.EntryDetailGRPCUnary,
			Path:         ar.Spec().Procedure,
			RemoteAddr:   remoteAddr,
			ForwardedFor: ar.Header().Get("X-Forwarded-For"),
			RequestID:    existingRequestID(ctx),
			Body:         ar.Any(),
		})
		if err != nil {
			return nil, err
		}

		childCtx := context.WithValue(sess.Context(), RequestIDKey, sess.RequestID())
		resp, err := next(childCtx, ar)

		var outcome auditing.Outcome
		outcome.Err = err
		if resp != nil {
			outcome.Body = resp.Any()
		}
		if ferr := sess.Finish(outcome); ferr != nil {
			i.logger.Error("unable to index", "error", ferr)
		}
		return resp, err
	}
}

func existingRequestID(ctx context.Context) string {
	if str, ok := ctx.Value(RequestIDKey).(string); ok {
		return str
	}
	return ""
}

// ConnectError is the serialized shape of a connect error in audit entries.
//
// Deprecated: produced by auditing.SerializableError; kept for compatibility.
type ConnectError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Err     string `json:"error"`
}

func (c ConnectError) Error() string {
	return fmt.Sprintf("%s (%d %s)", c.Err, c.Code, c.Message)
}
