//go:build integration
// +build integration

package auditing_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/metal-stack/metal-lib/auditing"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap/zaptest"
)

type ConnectionDetails struct {
	Port     string
	Host     string
	DB       string
	User     string
	Password string
}

func StartMeilisearch(t testing.TB) (container testcontainers.Container, c *ConnectionDetails, err error) {
	meilisearchMasterKey := "meili"

	ctx := context.Background()
	var log testcontainers.Logging
	if t != nil {
		log = testcontainers.TestLogger(t)
	}

	meiliContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "getmeili/meilisearch:v1.2.0",
			ExposedPorts: []string{"7700/tcp"},
			Env: map[string]string{
				"MEILI_MASTER_KEY":   meilisearchMasterKey,
				"MEILI_NO_ANALYTICS": "true",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("7700/tcp"),
			),
		},
		Started: true,
		Logger:  log,
	})
	if err != nil {
		panic(err.Error())
	}

	host, err := meiliContainer.Host(ctx)
	if err != nil {
		return meiliContainer, nil, err
	}
	port, err := meiliContainer.MappedPort(ctx, "7700")
	if err != nil {
		return meiliContainer, nil, err
	}

	conn := &ConnectionDetails{
		Host:     host,
		Port:     port.Port(),
		Password: meilisearchMasterKey,
	}

	return meiliContainer, conn, err
}

func TestAuditing_Meilisearch(t *testing.T) {
	container, c, err := StartMeilisearch(t)
	require.NoError(t, err)
	defer func() {
		err := container.Terminate(context.Background())
		require.NoError(t, err)
	}()

	var (
		url = "http://" + c.Host + ":" + c.Port
	)

	a, err := auditing.New(auditing.Config{
		URL:         url,
		APIKey:      c.Password,
		Log:         zaptest.NewLogger(t).Sugar(),
		IndexPrefix: fmt.Sprintf("test-%s", t.Name()),
	})
	require.NoError(t, err)

	entries, err := a.Search(auditing.EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 0)

	err = a.Index(auditing.Entry{
		Body: "test",
	})
	require.NoError(t, err)
	err = a.Flush()
	require.NoError(t, err)

	entries, err = a.Search(auditing.EntryFilter{
		Body: "test",
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	err = a.Index(auditing.Entry{
		RequestId:    "<full-request-id>",
		Type:         auditing.EntryTypeHTTP,
		User:         "admin",
		Tenant:       "global",
		Detail:       "POST",
		Phase:        auditing.EntryPhaseRequest,
		Path:         "/meilisearch",
		ForwardedFor: "127.0.0.1",
		RemoteAddr:   "10.0.0.1",
		Body:         "I want to pass",
	})
	require.NoError(t, err)
	err = a.Flush()
	require.NoError(t, err)

	entries, err = a.Search(auditing.EntryFilter{
		RequestId: "<full-request-id>",
		Body:      "I want to pass",
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entries, err = a.Search(auditing.EntryFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	err = a.Index(auditing.Entry{
		RequestId:    "<full-request-id>",
		Type:         auditing.EntryTypeHTTP,
		User:         "admin",
		Tenant:       "global",
		Detail:       "POST",
		Phase:        auditing.EntryPhaseResponse,
		Path:         "/meilisearch",
		ForwardedFor: "127.0.0.1",
		RemoteAddr:   "10.0.0.1",
		StatusCode:   418,
		Error:        fmt.Errorf("teapots cannot pass"),
		Body:         "you shall not pass",
	})
	require.NoError(t, err)
	err = a.Flush()
	require.NoError(t, err)

	entries, err = a.Search(auditing.EntryFilter{
		RequestId: "<full-request-id>",
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)

	respEntry := entries[0]
	require.Equal(t, auditing.EntryPhaseResponse, respEntry.Phase, "response entry should be first")
	require.Equal(t, "<full-request-id>", respEntry.RequestId)
	require.Equal(t, auditing.EntryTypeHTTP, respEntry.Type)
	require.Equal(t, "admin", respEntry.User)
	require.Equal(t, "global", respEntry.Tenant)
	require.Equal(t, auditing.EntryDetail("POST"), respEntry.Detail)
	require.Equal(t, "/meilisearch", respEntry.Path)
	require.Equal(t, 418, respEntry.StatusCode)
	require.Equal(t, "you shall not pass", respEntry.Body)
	require.EqualError(t, respEntry.Error, "teapots cannot pass")

	reqEntry := entries[1]
	require.Equal(t, auditing.EntryPhaseRequest, reqEntry.Phase, "response entry should be first")
	require.Equal(t, "<full-request-id>", reqEntry.RequestId)
	require.Equal(t, auditing.EntryTypeHTTP, reqEntry.Type)
	require.Equal(t, "admin", reqEntry.User)
	require.Equal(t, "global", reqEntry.Tenant)
	require.Equal(t, auditing.EntryDetail("POST"), reqEntry.Detail)
	require.Equal(t, "/meilisearch", reqEntry.Path)
}
