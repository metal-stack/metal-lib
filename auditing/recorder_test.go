package auditing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type recordingBackend struct {
	mu       sync.Mutex
	entries  []Entry
	indexErr error
}

func (b *recordingBackend) Index(e Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.indexErr != nil {
		return b.indexErr
	}
	b.entries = append(b.entries, e)
	return nil
}

func (b *recordingBackend) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.entries, nil
}

func (b *recordingBackend) setIndexErr(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.indexErr = err
}

func (b *recordingBackend) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

func (b *recordingBackend) ServiceName() string {
	return "recording"
}

func (b *recordingBackend) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	return healthstatus.HealthResult{Status: healthstatus.HealthStatusHealthy}, nil
}

func newTestRecorder(t *testing.T) (*Recorder, *recordingBackend) {
	t.Helper()
	backend := &recordingBackend{}
	rec, err := NewRecorder(backend, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return rec, backend
}

func Test_NewRecorder_NilBackend(t *testing.T) {
	_, err := NewRecorder(nil, slog.Default())
	require.Error(t, err)
}

func Test_Recorder_UnaryFlow(t *testing.T) {
	rec, backend := newTestRecorder(t)

	ctx := PutUserInContext(context.Background(), &User{
		Subject: "sub-1",
		Tenant:  "tenant-a",
		Project: "project-b",
	})

	sess, err := rec.Begin(ctx, Transport{
		Kind:   EntryTypeGRPC,
		Detail: EntryDetailGRPCUnary,
		Path:   "/api.v1.Foo/Bar",
		Body:   map[string]any{"in": true},
	})
	require.NoError(t, err)

	respBody := map[string]any{"out": true}
	require.NoError(t, sess.Finish(Outcome{Body: respBody}))

	entries, err := backend.Search(context.Background(), EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	req, resp := entries[0], entries[1]

	// request phase
	assert.Equal(t, EntryPhaseRequest, req.Phase)
	assert.Equal(t, EntryTypeGRPC, req.Type)
	assert.Equal(t, EntryDetailGRPCUnary, req.Detail)
	assert.Equal(t, "/api.v1.Foo/Bar", req.Path)
	assert.Equal(t, map[string]any{"in": true}, req.Body)
	assert.Nil(t, req.StatusCode)

	// both phases share the generated request id (uuid v7)
	id, err := uuid.Parse(req.RequestId)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
	assert.Equal(t, req.RequestId, resp.RequestId)

	// actor mapping
	assert.Equal(t, "sub-1", req.User)
	assert.Equal(t, "tenant-a", req.Tenant)
	assert.Equal(t, "project-b", req.Project)
	assert.Equal(t, "sub-1", resp.User)

	// response phase
	assert.Equal(t, EntryPhaseResponse, resp.Phase)
	assert.Equal(t, map[string]any{"out": true}, resp.Body)
	require.NotNil(t, resp.StatusCode)
	assert.Equal(t, 0, *resp.StatusCode) // success derives to 0
	assert.True(t, !resp.Timestamp.Before(req.Timestamp))
}

func Test_Recorder_StreamFlow(t *testing.T) {
	rec, backend := newTestRecorder(t)

	sess, err := rec.Begin(context.Background(), Transport{
		Kind:   EntryTypeGRPC,
		Detail: EntryDetailGRPCStream,
		Path:   "/api.v1.Foo/Stream",
	})
	require.NoError(t, err)
	require.NoError(t, sess.Finish(Outcome{Err: status.Error(codes.NotFound, "nope")}))

	entries, err := backend.Search(context.Background(), EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, EntryPhaseOpened, entries[0].Phase)
	assert.Equal(t, EntryPhaseClosed, entries[1].Phase)

	require.NotNil(t, entries[1].StatusCode)
	assert.Equal(t, int(codes.NotFound), *entries[1].StatusCode)
}

func Test_Recorder_ReusesRequestID(t *testing.T) {
	rec, backend := newTestRecorder(t)

	sess, err := rec.Begin(context.Background(), Transport{
		Kind:      EntryTypeHTTP,
		Detail:    EntryDetail(http.MethodPost),
		Path:      "/v1/x",
		RequestID: "existing-id",
	})
	require.NoError(t, err)
	require.Equal(t, "existing-id", sess.RequestID())
	require.NoError(t, sess.Finish(Outcome{StatusCode: 200}))

	entries, err := backend.Search(context.Background(), EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "existing-id", entries[0].RequestId)
	assert.Equal(t, "existing-id", entries[1].RequestId)
}

func Test_Recorder_AnonymousUser(t *testing.T) {
	rec, backend := newTestRecorder(t)

	sess, err := rec.Begin(context.Background(), Transport{Kind: EntryTypeHTTP, Detail: EntryDetail(http.MethodGet), Path: "/"})
	require.NoError(t, err)
	require.NoError(t, sess.Finish(Outcome{StatusCode: 200}))

	entries, err := backend.Search(context.Background(), EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// identity is the subject; anonymous has none
	assert.Empty(t, entries[0].User)
	assert.Empty(t, entries[0].Tenant)
	assert.Empty(t, entries[0].Project)
}

func Test_Recorder_StatusDerivation(t *testing.T) {
	tests := []struct {
		name     string
		kind     EntryType
		outcome  Outcome
		wantCode int
	}{
		{name: "success derives to zero", kind: EntryTypeGRPC, outcome: Outcome{}, wantCode: 0},
		{name: "explicit code wins", kind: EntryTypeGRPC, outcome: Outcome{StatusCode: 503}, wantCode: 503},
		{name: "grpc status error", kind: EntryTypeGRPC, outcome: Outcome{Err: status.Error(codes.InvalidArgument, "bad")}, wantCode: int(codes.InvalidArgument)},
		{name: "connect error", kind: EntryTypeGRPC, outcome: Outcome{Err: connect.NewError(connect.CodeAlreadyExists, errors.New("dup"))}, wantCode: int(connect.CodeAlreadyExists)},
		{name: "unrecognized rpc error", kind: EntryTypeGRPC, outcome: Outcome{Err: errors.New("plain")}, wantCode: int(codes.Unknown)},
		{name: "http explicit status", kind: EntryTypeHTTP, outcome: Outcome{StatusCode: 404, Err: errors.New("nf")}, wantCode: 404},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, backend := newTestRecorder(t)

			sess, err := rec.Begin(context.Background(), Transport{Kind: tt.kind, Detail: EntryDetailGRPCUnary, Path: "/x"})
			require.NoError(t, err)
			require.NoError(t, sess.Finish(tt.outcome))

			entries, err := backend.Search(context.Background(), EntryFilter{})
			require.NoError(t, err)
			require.Len(t, entries, 2)
			require.NotNil(t, entries[1].StatusCode)
			assert.Equal(t, tt.wantCode, *entries[1].StatusCode)
		})
	}
}

func Test_Recorder_FinishIsIdempotent(t *testing.T) {
	rec, backend := newTestRecorder(t)

	sess, err := rec.Begin(context.Background(), Transport{Kind: EntryTypeHTTP, Detail: EntryDetail(http.MethodPost), Path: "/x"})
	require.NoError(t, err)

	require.NoError(t, sess.Finish(Outcome{StatusCode: 200}))
	require.NoError(t, sess.Finish(Outcome{StatusCode: 500})) // second call is a no-op

	assert.Equal(t, 2, backend.count())
}

func Test_Recorder_BeginFailure(t *testing.T) {
	rec, backend := newTestRecorder(t)
	backend.setIndexErr(errors.New("backend down"))

	_, err := rec.Begin(context.Background(), Transport{Kind: EntryTypeHTTP, Detail: EntryDetail(http.MethodPost), Path: "/x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to index audit trace")
	assert.Equal(t, 0, backend.count())
}

func Test_Recorder_FinishFailure(t *testing.T) {
	rec, backend := newTestRecorder(t)

	sess, err := rec.Begin(context.Background(), Transport{Kind: EntryTypeHTTP, Detail: EntryDetail(http.MethodPost), Path: "/x"})
	require.NoError(t, err)

	backend.setIndexErr(errors.New("backend down"))
	err = sess.Finish(Outcome{StatusCode: 200})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to index audit trace")
}

func Test_SerializableError(t *testing.T) {
	tests := []struct {
		name string
		kind EntryType
		err  error
		want map[string]any
	}{
		{
			name: "nil error",
			kind: EntryTypeHTTP,
			err:  nil,
			want: nil,
		},
		{
			name: "http error keeps its shape",
			kind: EntryTypeHTTP,
			err:  httperrors.NewHTTPError(http.StatusConflict, fmt.Errorf("already exists")),
			want: map[string]any{"statuscode": float64(409), "message": "already exists"},
		},
		{
			name: "http plain error falls back to generic",
			kind: EntryTypeHTTP,
			err:  fmt.Errorf("boom"),
			want: map[string]any{"error": "boom"},
		},
		{
			name: "grpc status error",
			kind: EntryTypeGRPC,
			err:  status.Error(codes.NotFound, "missing"),
			want: map[string]any{"code": float64(5), "message": "NotFound", "error": "missing"},
		},
		{
			// legacy shape: the connect error string includes the code prefix
			name: "connect error",
			kind: EntryTypeGRPC,
			err:  connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("already exists")),
			want: map[string]any{"code": float64(6), "message": "already_exists", "error": "already_exists: already exists"},
		},
		{
			name: "grpc plain error falls back to generic",
			kind: EntryTypeGRPC,
			err:  fmt.Errorf("boom"),
			want: map[string]any{"error": "boom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SerializableError(tt.kind, tt.err)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}

			raw, err := json.Marshal(got)
			require.NoError(t, err)

			var parsed map[string]any
			require.NoError(t, json.Unmarshal(raw, &parsed))
			assert.Equal(t, tt.want, parsed)
		})
	}
}

func Test_SerializableError_Wrapping(t *testing.T) {
	// wrapped errors must resolve to their protocol shape
	wrapped := fmt.Errorf("outer: %w", httperrors.NewHTTPError(http.StatusBadRequest, fmt.Errorf("inner")))
	// legacy contract: the http shape is returned as a value
	got, ok := SerializableError(EntryTypeHTTP, wrapped).(httperrors.HTTPErrorResponse)
	require.True(t, ok, "expected httperrors.HTTPErrorResponse, got %T", got)
	assert.Equal(t, http.StatusBadRequest, got.StatusCode)

	// legacy quirk preserved: for a wrapped status error, status.FromError
	// yields the correct code but Message() is the full wrapper string
	wrappedRPC := fmt.Errorf("outer: %w", status.Error(codes.PermissionDenied, "denied"))
	raw, err := json.Marshal(SerializableError(EntryTypeGRPC, wrappedRPC))
	require.NoError(t, err)
	assert.JSONEq(t, `{"code": 7, "message": "PermissionDenied", "error": "outer: rpc error: code = PermissionDenied desc = denied"}`, string(raw))
}
