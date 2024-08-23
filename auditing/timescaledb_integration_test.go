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
				Detail:       "POST",
				Phase:        EntryPhaseResponse,
				Path:         "/v1/test/0",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.0",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000000",
				StatusCode:   200,
				Error:        "",
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000001",
				Type:         EntryTypeHTTP,
				Timestamp:    now.Add(1 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Detail:       "POST",
				Phase:        EntryPhaseResponse,
				Path:         "/v1/test/1",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.1",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000001",
				StatusCode:   201,
				Error:        "",
			},
			{
				Component:    "auditing.test",
				RequestId:    "00000000-0000-0000-0000-000000000002",
				Type:         EntryTypeHTTP,
				Timestamp:    now.Add(2 * time.Second),
				User:         "admin",
				Tenant:       "global",
				Detail:       "POST",
				Phase:        EntryPhaseRequest,
				Path:         "/v1/test/2",
				ForwardedFor: "127.0.0.1",
				RemoteAddr:   "10.0.0.2",
				Body:         "This is the body of 00000000-0000-0000-0000-000000000002",
				StatusCode:   0,
				Error:        "",
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
				entries, err := a.Search(EntryFilter{})
				require.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		{
			name: "insert one entry",
			t: func(t *testing.T, a Auditing) {
				err := a.Index(Entry{
					Body: "test",
				})
				require.NoError(t, err)
				err = a.Flush()
				require.NoError(t, err)

				entries, err := a.Search(EntryFilter{
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

				entries, err := a.Search(EntryFilter{})
				require.NoError(t, err)
				assert.Len(t, entries, len(es))

				sort.Slice(entries, func(i, j int) bool { return entries[i].RequestId < entries[j].RequestId })

				if diff := cmp.Diff(entries, es, cmpopts.IgnoreFields(Entry{}, "Id"), timeComparer); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}

				entries, err = a.Search(EntryFilter{
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

				entries, err := a.Search(EntryFilter{
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

				entries, err := a.Search(EntryFilter{
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
		// {
		// 	name: "filter on body",
		// 	t: func(t *testing.T, a Auditing) {
		// 		es := testEntries()
		// 		for _, e := range es {
		// 			err := a.Index(e)
		// 			require.NoError(t, err)
		// 		}

		// 		err := a.Flush()
		// 		require.NoError(t, err)

		// 		entries, err := a.Search(EntryFilter{
		// 			Body: fmt.Sprintf("%q", es[0].Body.(string)),
		// 		})
		// 		require.NoError(t, err)
		// 		require.Len(t, entries, 1)

		// 		if diff := cmp.Diff(entries[0], es[0]); diff != "" {
		// 			t.Errorf("diff (+got -want):\n %s", diff)
		// 		}
		// 	},
		// },
	}
	for i, tt := range tests {
		tt := tt

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
