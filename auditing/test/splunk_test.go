package test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/auditing/splunk"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_splunkAuditing_Index(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		entry auditing.Entry
		want  splunk.SplunkEvent
	}{
		{
			name: "index some entry with async",
			entry: auditing.Entry{
				Component:    "entry-component",
				RequestId:    "request-id",
				Type:         "entry-type",
				Timestamp:    now,
				User:         "entry-user",
				Tenant:       "entry-tenant",
				Detail:       "entry-detail",
				Phase:        "entry-phase",
				Path:         "entry-path",
				ForwardedFor: "entry-forwarded",
				RemoteAddr:   "entry-remote-addr",
				Body:         nil,
				StatusCode:   new(200),
				Error:        nil,
			},
			want: splunk.SplunkEvent{
				Time:       now.Unix(),
				Host:       "test-host",
				Source:     "metal-lib",
				SourceType: "_json",
				Index:      "test-index",
				Event: auditing.Entry{
					Component:    "entry-component",
					RequestId:    "request-id",
					Type:         "entry-type",
					Timestamp:    now,
					User:         "entry-user",
					Tenant:       "entry-tenant",
					Detail:       "entry-detail",
					Phase:        "entry-phase",
					Path:         "entry-path",
					ForwardedFor: "entry-forwarded",
					RemoteAddr:   "entry-remote-addr",
					Body:         nil,
					StatusCode:   new(200),
					Error:        nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/services/collector", func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var data splunk.SplunkEvent
				err = json.Unmarshal(body, &data)
				assert.NoError(t, err)

				if diff := cmp.Diff(data, tt.want); diff != "" {
					t.Errorf("diff = %s", diff)
				}

				w.WriteHeader(http.StatusOK)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			a, err := splunk.NewSplunk(auditing.Config{
				Component: "metal-lib",
				Log:       slog.Default(),
			}, splunk.SplunkConfig{
				Endpoint: server.URL,
				HECToken: "test-hec",
				Index:    "test-index",
				Host:     "test-host",
			})
			require.NoError(t, err)

			err = a.Index(tt.entry)
			require.NoError(t, err)
		})
	}
}

func Test_splunkAuditing_Health(t *testing.T) {
	tests := []struct {
		name string
		want healthstatus.HealthResult
	}{
		{
			name: "healthy",
			want: healthstatus.HealthResult{
				Status:  healthstatus.HealthStatusHealthy,
				Message: `audit backend "splunk" is healthy: HEC is healthy`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/services/collector/health", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"text":"HEC is healthy","code":17}`))
				w.WriteHeader(http.StatusOK)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			a, err := splunk.NewSplunk(auditing.Config{
				Component: "metal-lib",
				Log:       slog.Default(),
			}, splunk.SplunkConfig{
				Endpoint: server.URL,
				HECToken: "test-hec",
				Index:    "test-index",
				Host:     "test-host",
			})
			require.NoError(t, err)

			got, err := a.Check(t.Context())
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
