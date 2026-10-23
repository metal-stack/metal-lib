//go:build integration

package test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/auditing/timescaledb"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestAuditing_TimescaleDB(t *testing.T) {
	container, db, aud := startTimescaleDB(t, auditing.Config{
		Log: slog.Default(),
	})
	defer func() {
		err := container.Terminate(context.Background())
		require.NoError(t, err)
	}()

	Run(t, Suite{
		Backend: func(t testing.TB) auditing.Auditing {
			return aud
		},
		Profile: Profile{Capabilities: []Capability{
			CapabilitySearch,
			CapabilityMultiWordBody,
		}},
		Cleanup: func() {
			db.MustExec("DELETE FROM traces;")
		},
		HealthMessage: `audit backend "timescaledb" is healthy`,
	})
}

func startTimescaleDB(t testing.TB, config auditing.Config) (testcontainers.Container, *sqlx.DB, auditing.Auditing) {
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:2.25.1-pg17",
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
		Cmd: []string{"postgres", "-c", "max_connections=200"},
	}

	ctx := t.Context()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	ip, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	// the container can report ready while postgres is still finishing its
	// startup; retry the backend construction, whose migrations are idempotent
	var (
		auditing   auditing.Auditing
		backendErr error
	)
	for attempt := 1; attempt <= 10; attempt++ {
		auditing, backendErr = timescaledb.NewTimescaleDB(config, timescaledb.TimescaleDbConfig{
			Host:     ip,
			Port:     port.Port(),
			DB:       "postgres",
			User:     "postgres",
			Password: "password",
		})
		if backendErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, backendErr)

	source := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", ip, port.Port(), "postgres", "postgres", "password")

	db, err := sqlx.Connect("postgres", source)
	require.NoError(t, err)

	return container, db, auditing
}
