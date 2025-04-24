//go:build integration
// +build integration

package auditing

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

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

	for i, tt := range tests(ctx) {
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
