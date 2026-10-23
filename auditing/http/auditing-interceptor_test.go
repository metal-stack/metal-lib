package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/auditing/memory"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEnv builds a restful container with the audit filter installed after
// a filter that puts a user and a request id into the context, mimicking the
// logging/auth middleware that runs upstream in production.
func newTestEnv(t *testing.T, backend auditing.Auditing) *httptest.Server {
	t.Helper()

	filter, err := HttpFilter(backend, slog.Default())
	require.NoError(t, err)

	upstream := func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		ctx := req.Request.Context()
		ctx = context.WithValue(ctx, rest.RequestIDKey, "correlated-id")
		ctx = auditing.PutUserInContext(ctx, &auditing.User{
			Subject: "sub-42",
			Tenant:  "tenant-a",
			Project: "project-b",
		})
		req.Request = req.Request.WithContext(ctx)
		chain.ProcessFilter(req, resp)
	}

	ws := new(restful.WebService)
	ws.Path("/v1")
	ws.Route(ws.POST("/things").Produces(restful.MIME_JSON).To(func(req *restful.Request, resp *restful.Response) {
		_ = resp.WriteHeaderAndEntity(http.StatusCreated, map[string]any{"created": true})
	}))
	ws.Route(ws.GET("/things/{id}").Produces(restful.MIME_JSON).To(func(req *restful.Request, resp *restful.Response) {
		_ = resp.WriteHeaderAndEntity(http.StatusOK, map[string]any{"id": req.PathParameter("id")})
	}))
	ws.Route(ws.GET("/included").Produces(restful.MIME_JSON).To(func(req *restful.Request, resp *restful.Response) {
		_ = resp.WriteHeaderAndEntity(http.StatusOK, map[string]any{"ok": true})
	}).Metadata(Include, true))
	ws.Route(ws.POST("/excluded").Produces(restful.MIME_JSON).To(func(req *restful.Request, resp *restful.Response) {
		_ = resp.WriteHeaderAndEntity(http.StatusNoContent, nil)
	}).Metadata(Exclude, true))

	container := restful.NewContainer()
	container.Filter(upstream)
	container.Filter(filter)
	container.Add(ws)

	return httptest.NewServer(container)
}

func post(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func searchAll(t *testing.T, a auditing.Auditing) []auditing.Entry {
	t.Helper()
	entries, err := a.Search(context.Background(), auditing.EntryFilter{})
	require.NoError(t, err)
	return entries
}

func Test_HttpFilter_PostIsAudited(t *testing.T) {
	backend, err := memory.NewMemory(auditing.Config{Log: slog.Default()}, memory.MemoryConfig{})
	require.NoError(t, err)
	server := newTestEnv(t, backend)
	defer server.Close()

	resp := post(t, server.URL+"/v1/things", `{"name":"thing"}`)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	entries := searchAll(t, backend)
	require.Len(t, entries, 2)

	req, res := entries[0], entries[1]
	assert.Equal(t, auditing.EntryPhaseRequest, req.Phase)
	assert.Equal(t, auditing.EntryPhaseResponse, res.Phase)

	// both phases share the correlated request id from upstream middleware
	assert.Equal(t, "correlated-id", req.RequestId)
	assert.Equal(t, req.RequestId, res.RequestId)

	assert.Equal(t, auditing.EntryTypeHTTP, req.Type)
	assert.Equal(t, auditing.EntryDetail(http.MethodPost), req.Detail)
	assert.Equal(t, "/v1/things", req.Path)

	// actor identity is the subject
	assert.Equal(t, "sub-42", req.User)
	assert.Equal(t, "tenant-a", req.Tenant)
	assert.Equal(t, "project-b", req.Project)

	// request body captured, response body and status recorded
	assert.Equal(t, map[string]any{"name": "thing"}, req.Body)
	require.NotNil(t, res.StatusCode)
	assert.Equal(t, http.StatusCreated, *res.StatusCode)
	assert.Equal(t, map[string]any{"created": true}, res.Body)
}

func Test_HttpFilter_GetSkippedUnlessIncluded(t *testing.T) {
	backend, err := memory.NewMemory(auditing.Config{Log: slog.Default()}, memory.MemoryConfig{})
	require.NoError(t, err)
	server := newTestEnv(t, backend)
	defer server.Close()

	resp := get(t, server.URL+"/v1/things/1")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, searchAll(t, backend))

	resp = get(t, server.URL+"/v1/included")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, searchAll(t, backend), 2)
}

func Test_HttpFilter_ExcludedRouteIsNotAudited(t *testing.T) {
	backend, err := memory.NewMemory(auditing.Config{Log: slog.Default()}, memory.MemoryConfig{})
	require.NoError(t, err)
	server := newTestEnv(t, backend)
	defer server.Close()

	resp := post(t, server.URL+"/v1/excluded", `{}`)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Empty(t, searchAll(t, backend))
}

type failingBackend struct{}

func (failingBackend) Index(auditing.Entry) error {
	return errors.New("backend down")
}

func (failingBackend) Search(context.Context, auditing.EntryFilter) ([]auditing.Entry, error) {
	return nil, nil
}

func (failingBackend) ServiceName() string {
	return "failing"
}

func (failingBackend) Check(context.Context) (healthstatus.HealthResult, error) {
	return healthstatus.HealthResult{}, nil
}

func Test_HttpFilter_IndexFailureFailsRequest(t *testing.T) {
	server := newTestEnv(t, failingBackend{})
	defer server.Close()

	resp := post(t, server.URL+"/v1/things", `{"name":"thing"}`)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
