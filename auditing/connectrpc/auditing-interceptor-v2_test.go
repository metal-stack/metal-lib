package grpc

import (
	"context"
	"io"
	"log/slog"
	"testing"

	connectv2 "connectrpc.com/connect/v2"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/auditing/memory"
	"github.com/stretchr/testify/require"
)

type testMsg struct {
	Text string
}

type recordingStream struct {
	recvErr error
	recvMsg any
	sent    []any
}

func (s *recordingStream) Receive(msg any) error {
	if s.recvErr != nil {
		return s.recvErr
	}
	if in, ok := msg.(*testMsg); ok {
		if src, ok := s.recvMsg.(*testMsg); ok {
			*in = *src
		}
	}
	return nil
}

func (s *recordingStream) SendHeaders() error { return nil }

func (s *recordingStream) Send(msg any) error {
	s.sent = append(s.sent, msg)
	return nil
}

func newTestInterceptor(t *testing.T, shouldAudit func(string) bool) (connectv2.ServerInterceptor, auditing.Auditing) {
	t.Helper()
	backend, err := memory.NewMemory(auditing.Config{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}, memory.MemoryConfig{})
	require.NoError(t, err)

	interceptor, err := NewConnectV2Interceptor(backend, slog.New(slog.NewTextHandler(io.Discard, nil)), shouldAudit)
	require.NoError(t, err)
	return interceptor, backend
}

func TestNewConnectV2InterceptorNilAuditing(t *testing.T) {
	_, err := NewConnectV2Interceptor(nil, slog.Default(), func(string) bool { return true })
	require.Error(t, err)
}

func TestNewConnectV2InterceptorUnary(t *testing.T) {
	interceptor, backend := newTestInterceptor(t, func(string) bool { return true })

	stream := &recordingStream{recvMsg: &testMsg{Text: "hello"}}
	next := func(ctx context.Context, spec connectv2.Spec, stream connectv2.ServerStream) error {
		req := &testMsg{}
		if err := stream.Receive(req); err != nil {
			return err
		}
		return stream.Send(&testMsg{Text: "resp:" + req.Text})
	}

	ctx := context.Background()
	err := interceptor(next)(ctx, connectv2.Spec{Procedure: "/test.Service/Unary", StreamType: connectv2.StreamTypeUnary}, stream)
	require.NoError(t, err)

	requests, err := backend.Search(ctx, auditing.EntryFilter{Phase: auditing.EntryPhaseRequest, Detail: auditing.EntryDetailGRPCUnary})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, &testMsg{Text: "hello"}, requests[0].Body)
	require.NotEmpty(t, requests[0].RequestId)

	responses, err := backend.Search(ctx, auditing.EntryFilter{Phase: auditing.EntryPhaseResponse, Detail: auditing.EntryDetailGRPCUnary})
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, &testMsg{Text: "resp:hello"}, responses[0].Body)
	require.Equal(t, new(int), responses[0].StatusCode)
}

func TestNewConnectV2InterceptorStreaming(t *testing.T) {
	interceptor, backend := newTestInterceptor(t, func(string) bool { return true })

	stream := &recordingStream{}
	next := func(ctx context.Context, spec connectv2.Spec, stream connectv2.ServerStream) error {
		return nil
	}

	ctx := context.Background()
	err := interceptor(next)(ctx, connectv2.Spec{Procedure: "/test.Service/Stream", StreamType: connectv2.StreamTypeServer}, stream)
	require.NoError(t, err)

	opened, err := backend.Search(ctx, auditing.EntryFilter{Phase: auditing.EntryPhaseOpened, Detail: auditing.EntryDetailGRPCStream})
	require.NoError(t, err)
	require.Len(t, opened, 1)

	closed, err := backend.Search(ctx, auditing.EntryFilter{Phase: auditing.EntryPhaseClosed, Detail: auditing.EntryDetailGRPCStream})
	require.NoError(t, err)
	require.Len(t, closed, 1)
	require.Equal(t, new(int), closed[0].StatusCode)
}

func TestNewConnectV2InterceptorError(t *testing.T) {
	interceptor, backend := newTestInterceptor(t, func(string) bool { return true })

	stream := &recordingStream{recvMsg: &testMsg{Text: "hello"}}
	next := func(ctx context.Context, spec connectv2.Spec, stream connectv2.ServerStream) error {
		return connectv2.NewError(connectv2.CodeNotFound, "not found")
	}

	ctx := context.Background()
	err := interceptor(next)(ctx, connectv2.Spec{Procedure: "/test.Service/Unary", StreamType: connectv2.StreamTypeUnary}, stream)
	require.Error(t, err)

	responses, err := backend.Search(ctx, auditing.EntryFilter{Phase: auditing.EntryPhaseResponse})
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, new(int(int(connectv2.CodeNotFound))), responses[0].StatusCode)
	require.NotNil(t, responses[0].Error)
}

func TestNewConnectV2InterceptorShouldAuditFalse(t *testing.T) {
	interceptor, backend := newTestInterceptor(t, func(string) bool { return false })

	called := false
	next := func(ctx context.Context, spec connectv2.Spec, stream connectv2.ServerStream) error {
		called = true
		return nil
	}

	ctx := context.Background()
	err := interceptor(next)(ctx, connectv2.Spec{Procedure: "/test.Service/Unary", StreamType: connectv2.StreamTypeUnary}, &recordingStream{})
	require.NoError(t, err)
	require.True(t, called)

	all, err := backend.Search(ctx, auditing.EntryFilter{})
	require.NoError(t, err)
	require.Empty(t, all)
}
