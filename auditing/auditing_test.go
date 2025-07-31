package auditing_test

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	now = time.Now().UTC()

	// postgres does not store the nano seconds, so we neglect them for comparison:
	timeComparer = cmp.Comparer(func(x, y time.Time) bool {
		return x.Unix() == y.Unix()
	})

	testEntries = func() []auditing.Entry {
		return []auditing.Entry{
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000000",
				Type:         auditing.EntryTypeHTTP,
				Timestamp:    now,
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        auditing.EntryPhaseResponse,
				Path:         "/v1/test/0",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.0",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000000",
				StatusCode:   pointer.Pointer(200),
				Error:        nil,
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000001",
				Type:         auditing.EntryTypeHTTP,
				Timestamp:    now.Add(1 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        auditing.EntryPhaseResponse,
				Path:         "/v1/test/1",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.1",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000001",
				StatusCode:   pointer.Pointer(201),
				Error:        nil,
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000002",
				Type:         auditing.EntryTypeHTTP,
				Timestamp:    now.Add(2 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        auditing.EntryPhaseRequest,
				Path:         "/v1/test/2",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.2",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000002",
				StatusCode:   nil,
				Error:        auditing.SerializableError(fmt.Errorf("an error")),
			},
		}
	}

	tests = func(ctx context.Context) []struct {
		name string
		t    func(t *testing.T, a auditing.Auditing)
	} {
		return []struct {
			name string
			t    func(t *testing.T, a auditing.Auditing)
		}{
			{
				name: "no entries, no search results",
				t: func(t *testing.T, a auditing.Auditing) {
					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					assert.Empty(t, entries)
				},
			},
			{
				name: "insert one entry",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Timestamp: now,
						Body:      "test",
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Body: "test",
					})
					require.NoError(t, err)
					assert.Len(t, entries, 1)
				},
			},
			{
				name: "insert a couple of entries",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					assert.Len(t, entries, len(es))

					sort.Slice(entries, func(i, j int) bool { return entries[i].RequestId < entries[j].RequestId })

					if diff := cmp.Diff(entries, es, cmpopts.IgnoreFields(auditing.Entry{}, "Id", "Error"), timeComparer); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}

					entries, err = a.Search(ctx, auditing.EntryFilter{
						Body: "This",
					})

					require.NoError(t, err)
					assert.Len(t, entries, len(es))
				},
			},
			{
				name: "filter search on component",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Component: "a",
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						From:      now.Add(-1 * time.Minute),
						To:        now.Add(1 * time.Minute),
						Component: "a",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "a",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on forwarded for",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						ForwardedFor: "a",
						RequestId:    "1",
						Timestamp:    now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						ForwardedFor: "a",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component:    "auditing.test",
						ForwardedFor: "a",
						RequestId:    "1",
						Timestamp:    now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on path",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Path:      "/a/b/c/d",
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Path: "/b/c", // partial match
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "auditing.test",
						Path:      "/a/b/c/d",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on status code",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						StatusCode: pointer.Pointer(400),
						RequestId:  "1",
						Timestamp:  now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						StatusCode: pointer.Pointer(400),
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component:  "auditing.test",
						StatusCode: pointer.Pointer(400),
						RequestId:  "1",
						Timestamp:  now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on status code with 0",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					err = a.Index(auditing.Entry{
						RequestId:  "2",
						Timestamp:  now,
						StatusCode: pointer.Pointer(0),
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						StatusCode: pointer.Pointer(0),
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component:  "auditing.test",
						StatusCode: pointer.Pointer(0),
						RequestId:  "2",
						Timestamp:  now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on tenant",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Tenant:    "a",
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Tenant: "a",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "auditing.test",
						Tenant:    "a",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on project",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Project:   "a",
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Project: "a",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "auditing.test",
						Project:   "a",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on user",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						User:      "a",
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						User: "a",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "auditing.test",
						User:      "a",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "define a limit higher than amount of results",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Limit: 1,
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[0]); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "define a limit equal the amount of results",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Limit: 3,
					})
					require.NoError(t, err)
					require.Len(t, entries, 3)

					if diff := cmp.Diff(entries, es, cmpopts.IgnoreFields(auditing.Entry{}, "Error")); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "define a limit smaller than amount of results",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						RequestId: "1",
						Timestamp: now,
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Limit: 5,
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], auditing.Entry{
						Component: "auditing.test",
						RequestId: "1",
						Timestamp: now,
					}); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on rqid",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						RequestId: es[0].RequestId,
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[0], cmpopts.IgnoreFields(auditing.Entry{}, "Id", "Error"), timeComparer); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter search on phase",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					var wantEntries []auditing.Entry
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)

						if e.Phase == auditing.EntryPhaseResponse {
							wantEntries = append(wantEntries, e)
						}
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Phase: auditing.EntryPhaseResponse,
					})
					require.NoError(t, err)
					require.Len(t, entries, 2)

					sort.Slice(entries, func(i, j int) bool { return entries[i].RequestId < entries[j].RequestId })

					if diff := cmp.Diff(entries, wantEntries, cmpopts.IgnoreFields(auditing.Entry{}, "Id", "Error"), timeComparer); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter on body missing one word",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Body: "This is body",
					})
					require.NoError(t, err)
					assert.Len(t, entries, len(es))

					entries, err = a.Search(ctx, auditing.EntryFilter{
						Body: `"This is body"`,
					})
					require.NoError(t, err)
					assert.Empty(t, entries)
				},
			},
			{
				name: "filter on body capital ignored",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Body: "this is the BODY",
					})
					require.NoError(t, err)
					assert.Len(t, entries, len(es))
				},
			},
			{
				name: "filter on body",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Body: fmt.Sprintf("%s", es[0].Body.(string)),
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[0]); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter on body partial words",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Body: "002",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[2], cmpopts.IgnoreFields(auditing.Entry{}, "Error")); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter on every filter field",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						Limit:        1,
						From:         now.Add(-1 * time.Minute),
						To:           now.Add(1 * time.Minute),
						Component:    "auditing.test",
						RequestId:    "00000000-0000-0000-0000-000000000000",
						Type:         "http",
						User:         "admin",
						Tenant:       "global",
						Project:      "project",
						Detail:       "POST",
						Phase:        "response",
						Path:         "/v1/test/0",
						ForwardedFor: "127.0.0.1",
						RemoteAddr:   "10.0.0.0",
						Body:         fmt.Sprintf("%s", es[0].Body.(string)),
						StatusCode:   pointer.Pointer(200),
						Error:        "",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[0]); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "filter on nothing",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					require.Len(t, entries, len(testEntries()))
				},
			},
			{
				name: "filter on query does not affect other filters",
				t: func(t *testing.T, a auditing.Auditing) {
					es := testEntries()
					for _, e := range es {
						err := a.Index(e)
						require.NoError(t, err)
					}

					entries, err := a.Search(ctx, auditing.EntryFilter{
						From:      now.Add(-1 * time.Minute),
						To:        now.Add(1 * time.Minute),
						RequestId: "00000000-0000-0000-0000-000000000000",
						Phase:     "response",
						Path:      "/v1/test/0",
						Body:      "00000000",
					})
					require.NoError(t, err)
					require.Len(t, entries, 1)

					if diff := cmp.Diff(entries[0], es[0]); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
					}
				},
			},
			{
				name: "fields are defaulted during indexing",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					assert.Len(t, entries, 1)
					assert.Equal(t, "auditing.test", entries[0].Component)
					assert.WithinDuration(t, time.Now(), entries[0].Timestamp, 1*time.Second)
				},
			},
			{
				name: "index an http error",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Error: auditing.SerializableError(httperrors.NewHTTPError(http.StatusConflict, fmt.Errorf("already exists"))),
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					assert.Len(t, entries, 1)
					assert.Equal(t, "auditing.test", entries[0].Component)
					assert.Equal(t, map[string]any{"statuscode": float64(409), "message": "already exists"}, entries[0].Error)
					assert.WithinDuration(t, time.Now(), entries[0].Timestamp, 1*time.Second)
				},
			},
			{
				name: "index a connect error",
				t: func(t *testing.T, a auditing.Auditing) {
					err := a.Index(auditing.Entry{
						Error: auditing.SerializableError(connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("already exists"))),
					})
					require.NoError(t, err)

					entries, err := a.Search(ctx, auditing.EntryFilter{})
					require.NoError(t, err)
					assert.Len(t, entries, 1)
					assert.Equal(t, "auditing.test", entries[0].Component)
					assert.Equal(t, map[string]any{"code": "already_exists", "error": "already_exists: already exists"}, entries[0].Error)
					assert.WithinDuration(t, time.Now(), entries[0].Timestamp, 1*time.Second)
				},
			},
		}
	}
)
