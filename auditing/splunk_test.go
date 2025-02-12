package auditing

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_splunkAuditing_Index(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		async        bool
		asyncRetry   int
		asyncTimeout time.Duration
		entry        Entry
		want         splunkEvent
	}{
		{
			name:         "index some entry with async",
			async:        true,
			asyncRetry:   0,
			asyncTimeout: 500 * time.Millisecond,
			entry: Entry{
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
				StatusCode:   0,
				Error:        nil,
			},
			want: splunkEvent{
				Time:       now.Unix(),
				Host:       "test-host",
				Source:     "metal-lib",
				SourceType: "_json",
				Index:      "test-index",
				Event: Entry{
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
					StatusCode:   0,
					Error:        nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := make(chan bool)
			defer close(received)

			mux := http.NewServeMux()
			mux.HandleFunc("/services/collector", func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var data splunkEvent
				err = json.Unmarshal(body, &data)
				assert.NoError(t, err)

				if diff := cmp.Diff(data, tt.want); diff != "" {
					t.Errorf("diff = %s", diff)
				}

				w.WriteHeader(http.StatusOK)

				received <- true
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			a, err := NewSplunk(Config{
				Component:    "metal-lib",
				Log:          slog.Default(),
				Async:        tt.async,
				AsyncRetry:   tt.asyncRetry,
				AsyncTimeout: tt.asyncTimeout,
			}, SplunkConfig{
				Endpoint: server.URL,
				HECToken: "test-hec",
				Index:    "test-index",
				Host:     "test-host",
			})
			require.NoError(t, err)

			err = a.Index(tt.entry)
			require.NoError(t, err)

			if tt.async {
				<-received
			}
		})
	}
}
