package rest

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewHealth(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.Nil(t, err)

	type args struct {
		log      *zap.Logger
		basePath string
		h        []HealthCheck
	}
	tests := []struct {
		name string
		args args
		want *status
	}{
		{
			name: "check without giving health checks",
			args: args{
				log:      logger,
				basePath: "/",
				h:        nil,
			},
			want: &status{
				Status:  HealthStatusHealthy,
				Message: "OK",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ws := NewHealth(tt.args.log, tt.args.basePath, tt.args.h...)

			container := restful.NewContainer().Add(ws)

			createReq := httptest.NewRequest("GET", "/v1/health", nil)
			w := httptest.NewRecorder()
			container.ServeHTTP(w, createReq)

			resp := w.Result()
			defer resp.Body.Close()
			var s status
			err = json.NewDecoder(resp.Body).Decode(&s)
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, &s); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
