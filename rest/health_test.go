package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

type succeedingCheck struct{}

func (e *succeedingCheck) ServiceName() string {
	return "success"
}

func (e *succeedingCheck) Check(ctx context.Context) (HealthStatus, error) {
	return HealthStatusHealthy, nil
}

type failingCheck struct{}

func (e *failingCheck) ServiceName() string {
	return "fail"
}

func (e *failingCheck) Check(ctx context.Context) (HealthStatus, error) {
	return HealthStatusUnhealthy, fmt.Errorf("facing an issue")
}

func TestNewHealth(t *testing.T) {
	logger := slog.Default()

	type args struct {
		log      *slog.Logger
		basePath string
		service  string
		h        []HealthCheck
	}
	tests := []struct {
		name string
		args args
		want *HealthResponse
	}{
		{
			name: "check without giving health checks",
			args: args{
				log:      logger,
				basePath: "/",
				h:        nil,
			},
			want: &HealthResponse{
				Status:   HealthStatusHealthy,
				Message:  "",
				Services: map[string]HealthResult{},
			},
		},
		{
			name: "check with one service error",
			args: args{
				log:      logger,
				basePath: "/",
				h:        []HealthCheck{&succeedingCheck{}, &failingCheck{}},
			},
			want: &HealthResponse{
				Status:  HealthStatusPartiallyUnhealthy,
				Message: "facing an issue",
				Services: map[string]HealthResult{
					"success": {
						Status:  HealthStatusHealthy,
						Message: "",
					},
					"fail": {
						Status:  HealthStatusUnhealthy,
						Message: "facing an issue",
					},
				},
			},
		},
		{
			name: "query specific service",
			args: args{
				log:      logger,
				basePath: "/",
				h:        []HealthCheck{&succeedingCheck{}, &failingCheck{}},
				service:  "success",
			},
			want: &HealthResponse{
				Status:  HealthStatusHealthy,
				Message: "",
				Services: map[string]HealthResult{
					"success": {
						Status:  HealthStatusHealthy,
						Message: "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ws, err := NewHealth(tt.args.log, tt.args.basePath, tt.args.h...)
			require.NoError(t, err)

			container := restful.NewContainer().Add(ws)

			path := "/v1/health"
			if tt.args.service != "" {
				path += "?service=" + tt.args.service
			}
			createReq := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			container.ServeHTTP(w, createReq)

			resp := w.Result()
			defer resp.Body.Close()
			var s HealthResponse
			err = json.NewDecoder(resp.Body).Decode(&s)
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, &s); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
