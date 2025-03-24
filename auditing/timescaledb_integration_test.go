//go:build integration
// +build integration

package auditing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestAuditing_TimescaleDB(t *testing.T) {
	ctx := context.Background()
	container, auditing := StartTimescaleDB(t, Config{
		Log: slog.Default(),
	})
	defer func() {
		err := container.Terminate(context.Background())
		require.NoError(t, err)
	}()

	now := time.Now().UTC()
	// postgres does not store the nano seconds, so we neglect them for comparison:
	timeComparer := cmp.Comparer(func(x, y time.Time) bool {
		return x.Unix() == y.Unix()
	})

	testEntries := func() []Entry {
		return []Entry{
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000000",
				Type:         EntryTypeHTTP,
				Timestamp:    now,
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        EntryPhaseResponse,
				Path:         "/v1/test/0",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.0",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000000",
				StatusCode:   200,
				Error:        nil,
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000001",
				Type:         EntryTypeHTTP,
				Timestamp:    now.Add(1 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        EntryPhaseResponse,
				Path:         "/v1/test/1",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.1",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000001",
				StatusCode:   201,
				Error:        nil,
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000002",
				Type:         EntryTypeHTTP,
				Timestamp:    now.Add(2 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Project:      "project",
				Detail:       "POST",
				Phase:        EntryPhaseRequest,
				Path:         "/v1/test/2",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.2",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000002",
				StatusCode:   0,
				Error:        "an error",
			},
		}
	}

	tests := []struct {
		name string
		t    func(t *testing.T, a Auditing)
	}{
		{
			name: "no entries, no search results",
			t: func(t *testing.T, a Auditing) {
				entries, err := a.Search(ctx, EntryFilter{})
				require.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		{
			name: "insert one entry",
			t: func(t *testing.T, a Auditing) {
				err := a.Index(Entry{
					Timestamp: now,
					Body:      "test",
				})
				require.NoError(t, err)
				err = a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					Body: "test",
				})
				require.NoError(t, err)
				assert.Len(t, entries, 1)
			},
		},
		{
			name: "insert a couple of entries",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{})
				require.NoError(t, err)
				assert.Len(t, entries, len(es))

				sort.Slice(entries, func(i, j int) bool { return entries[i].RequestId < entries[j].RequestId })

				if diff := cmp.Diff(entries, es, cmpopts.IgnoreFields(Entry{}, "Id"), timeComparer); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}

				entries, err = a.Search(ctx, EntryFilter{
					Body: "This",
				})

				require.NoError(t, err)
				assert.Len(t, entries, len(es))
			},
		},
		{
			name: "filter search on rqid",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					RequestId: es[0].RequestId,
				})
				require.NoError(t, err)
				require.Len(t, entries, 1)

				if diff := cmp.Diff(entries[0], es[0], cmpopts.IgnoreFields(Entry{}, "Id"), timeComparer); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}
			},
		},
		{
			name: "filter search on phase",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				var wantEntries []Entry
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)

					if e.Phase == EntryPhaseResponse {
						wantEntries = append(wantEntries, e)
					}
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					Phase: EntryPhaseResponse,
				})
				require.NoError(t, err)
				require.Len(t, entries, 2)

				sort.Slice(entries, func(i, j int) bool { return entries[i].RequestId < entries[j].RequestId })

				if diff := cmp.Diff(entries, wantEntries, cmpopts.IgnoreFields(Entry{}, "Id"), timeComparer); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}
			},
		},
		{
			name: "filter on body missing one word",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					Body: "This is body",
				})
				require.NoError(t, err)
				assert.Len(t, entries, len(es))

				entries, err = a.Search(ctx, EntryFilter{
					Body: `"This is body"`,
				})
				require.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		{
			name: "filter on body capital ignored",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					Body: "this is the BODY",
				})
				require.NoError(t, err)
				assert.Len(t, entries, len(es))
			},
		},
		{
			name: "filter on body",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
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
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					Body: fmt.Sprintf("002"),
				})
				require.NoError(t, err)
				require.Len(t, entries, 1)

				if diff := cmp.Diff(entries[0], es[2]); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}
			},
		},
		{
			name: "filter on every filter field",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
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
					StatusCode:   200,
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
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{})
				require.NoError(t, err)
				require.Len(t, entries, len(testEntries()))
			},
		},
		{
			name: "filter on query does not affect other filters",
			t: func(t *testing.T, a Auditing) {
				es := testEntries()
				for _, e := range es {
					err := a.Index(e)
					require.NoError(t, err)
				}

				err := a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
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
			t: func(t *testing.T, a Auditing) {
				err := a.Index(Entry{})
				require.NoError(t, err)

				err = a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{})
				require.NoError(t, err)
				assert.Len(t, entries, 1)
				assert.Equal(t, "auditing.test", entries[0].Component)
				assert.WithinDuration(t, time.Now(), entries[0].Timestamp, 1*time.Second)
			},
		},
		{
			name: "backwards compatibility with old error type",
			t: func(t *testing.T, a Auditing) {
				err := a.Index(Entry{
					RequestId: "1",
					Timestamp: now,
					Error:     fmt.Errorf("an error"),
				})
				require.NoError(t, err)

				err = a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(ctx, EntryFilter{
					From:      now.Add(-1 * time.Minute),
					To:        now.Add(1 * time.Minute),
					RequestId: "1",
				})
				require.NoError(t, err)
				require.Len(t, entries, 1)

				if diff := cmp.Diff(entries[0], Entry{
					RequestId: "1",
					Timestamp: now,
					Error:     map[string]any{}, // unfortunately this was a regression and the error was marshalled as an empty map because error does export any fields
				}); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d %s", i, tt.name), func(t *testing.T) {
			defer func() {
				a := auditing.(*timescaleAuditing)
				a.db.MustExec("DELETE FROM traces;")
			}()

			tt.t(t, auditing)
		})
	}
}

func StartTimescaleDB(t testing.TB, config Config) (testcontainers.Container, Auditing) {
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:2.16.1-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
		Cmd: []string{"postgres", "-c", "max_connections=200"},
	}

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	ip, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	auditing, err := NewTimescaleDB(config, TimescaleDbConfig{
		Host:     ip,
		Port:     port.Port(),
		DB:       "postgres",
		User:     "postgres",
		Password: "password",
	})
	require.NoError(t, err)

	return container, auditing
}
