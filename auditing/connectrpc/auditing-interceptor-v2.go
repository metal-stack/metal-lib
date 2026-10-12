package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	connectv2 "connectrpc.com/connect/v2"
	"github.com/metal-stack/metal-lib/auditing"
)

// NewConnectV2Interceptor creates a connect-go v2 server interceptor that
// audits every RPC matching shouldAudit.
func NewConnectV2Interceptor(a auditing.Auditing, logger *slog.Logger, shouldAudit func(fullMethod string) bool) (connectv2.ServerInterceptor, error) {
	if a == nil {
		return nil, fmt.Errorf("cannot use nil auditing to create connect interceptor")
	}
	i := auditingConnectInterceptor{
		auditing:    a,
		logger:      logger,
		shouldAudit: shouldAudit,
	}
	return i.WrapServer, nil
}

// WrapServer implements a connect-go v2 connect.ServerInterceptor.
func (a auditingConnectInterceptor) WrapServer(next connectv2.ServerFunc) connectv2.ServerFunc {
	return func(ctx context.Context, spec connectv2.Spec, stream connectv2.ServerStream) error {
		if !a.shouldAudit(spec.Procedure) {
			return next(ctx, spec, stream)
		}

		requestID, err := requestIDFromContext(ctx)
		if err != nil {
			return err
		}
		childCtx := context.WithValue(ctx, RequestIDKey, requestID)

		isUnary := spec.StreamType == connectv2.StreamTypeUnary

		base := a.baseEntry(ctx, requestID, spec.Procedure)
		if info, ok := connectv2.CallInfoForServerContext(ctx); ok {
			setRemoteAddr(&base, info.RequestHeader().Get, info.PeerAddr)
		}

		var responseBody any
		auditStream := &auditingServerStreamV2{ServerStream: stream}

		if isUnary {
			base.Detail = auditing.EntryDetailGRPCUnary
			base.Phase = auditing.EntryPhaseRequest
			auditStream.onReceive = func(msg any) {
				entry := base
				entry.Body = msg
				if err := a.auditing.Index(entry); err != nil {
					a.logger.Error("unable to index", "error", err)
				}
			}
			auditStream.onSend = func(msg any) {
				responseBody = msg
			}
		} else {
			base.Detail = auditing.EntryDetailGRPCStream
			base.Phase = auditing.EntryPhaseOpened
			if err := a.auditing.Index(base); err != nil {
				a.logger.Error("unable to index", "error", err)
			}
		}

		err = next(childCtx, spec, auditStream)

		respEntry := base
		respEntry.Id = ""
		respEntry.Timestamp = time.Now()
		respEntry.Body = nil
		respEntry.Error = nil
		if isUnary {
			respEntry.Phase = auditing.EntryPhaseResponse
			if err == nil {
				respEntry.Body = responseBody
			}
		} else {
			respEntry.Phase = auditing.EntryPhaseClosed
		}
		respEntry.StatusCode = statusCode(err)
		if err != nil {
			respEntry.Error = serializableError(err)
		}

		if indexErr := a.auditing.Index(respEntry); indexErr != nil {
			a.logger.Error("unable to index", "error", indexErr)
		}

		return err
	}
}

// auditingServerStreamV2 wraps a connect-go v2 connect.ServerStream to observe
// the messages flowing through a unary RPC.
type auditingServerStreamV2 struct {
	connectv2.ServerStream
	onReceive func(any)
	onSend    func(any)
}

func (s *auditingServerStreamV2) Receive(msg any) error {
	err := s.ServerStream.Receive(msg)
	if err == nil && s.onReceive != nil {
		s.onReceive(msg)
	}
	return err
}

func (s *auditingServerStreamV2) Send(msg any) error {
	err := s.ServerStream.Send(msg)
	if err == nil && s.onSend != nil {
		s.onSend(msg)
	}
	return err
}
