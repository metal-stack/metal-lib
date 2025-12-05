package connect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
)

type auditingConnectInterceptor struct {
	auditing    auditing.Auditing
	logger      *slog.Logger
	shouldAudit func(fullMethod string) bool
}

func NewInterceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connect.Interceptor, error) {
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

		auditReqContext := auditing.Entry{
			Timestamp: time.Now(),
			RequestId: requestID,
			Detail:    auditing.EntryDetailGRPCStream,
			Path:      s.Procedure,
			Phase:     auditing.EntryPhaseOpened,
			Type:      auditing.EntryTypeGRPC,
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

		auditReqContext.PrepareForNextPhase()
		scc := next(childCtx, s)

		auditReqContext.Phase = auditing.EntryPhaseClosed
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

		auditReqContext := auditing.Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       auditing.EntryDetailGRPCStream,
			Path:         shc.Spec().Procedure,
			Phase:        auditing.EntryPhaseOpened,
			Type:         auditing.EntryTypeGRPC,
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

		auditReqContext.PrepareForNextPhase()
		err = next(childCtx, shc)
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = auditing.SerializableError(err)

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

		auditReqContext := auditing.Entry{
			Timestamp:    time.Now(),
			RequestId:    requestID,
			Detail:       auditing.EntryDetailGRPCUnary,
			Path:         ar.Spec().Procedure,
			Phase:        auditing.EntryPhaseRequest,
			Type:         auditing.EntryTypeGRPC,
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

		auditReqContext.PrepareForNextPhase()

		resp, err := next(childCtx, ar)

		auditReqContext.Phase = auditing.EntryPhaseResponse
		auditReqContext.StatusCode = statusCodeFromGrpcOrConnect(err)

		if err != nil {
			auditReqContext.Error = auditing.SerializableError(err)

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
