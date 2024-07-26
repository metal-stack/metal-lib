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
	"github.com/metal-stack/metal-lib/healthstatus"
	"github.com/stretchr/testify/require"
)

type succeedingCheck struct{}

func (e *succeedingCheck) ServiceName() string {
	return "success"
}

func (e *succeedingCheck) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	return healthstatus.HealthResult{
		Message: "",
		Services: map[string]healthstatus.HealthResult{
			"successPartition": {
				Status:  healthstatus.HealthStatusHealthy,
				Message: "",
			},
		},
	}, nil
}

type failingCheck struct{}

func (e *failingCheck) ServiceName() string {
	return "fail"
}

func (e *failingCheck) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	return healthstatus.HealthResult{
		Message: "",
		Services: map[string]healthstatus.HealthResult{
			"failPartition": {
				Status:  healthstatus.HealthStatusUnhealthy,
				Message: "facing an issue",
			},
		},
	}, fmt.Errorf("facing an issue")
}

func TestNewHealth(t *testing.T) {
	logger := slog.Default()

	type args struct {
		log      *slog.Logger
		basePath string
		service  string
		h        []healthstatus.HealthCheck
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
				Status:  healthstatus.HealthStatusHealthy,
				Message: "",
			},
		},
		{
			name: "check with one service error",
			args: args{
				log:      logger,
				basePath: "/",
				h:        []healthstatus.HealthCheck{&succeedingCheck{}, &failingCheck{}},
			},
			want: &HealthResponse{
				Status:  healthstatus.HealthStatusPartiallyUnhealthy,
				Message: "facing an issue",
				Services: map[string]HealthResponse{
					"success": {
						Status:  healthstatus.HealthStatusHealthy,
						Message: "",
						Services: map[string]HealthResponse{
							"successPartition": {
								Status:  healthstatus.HealthStatusHealthy,
								Message: "",
							},
						},
					},
					"fail": {
						Status:  healthstatus.HealthStatusUnhealthy,
						Message: "facing an issue",
						Services: map[string]HealthResponse{
							"failPartition": {
								Status:  healthstatus.HealthStatusUnhealthy,
								Message: "facing an issue",
							},
						},
					},
				},
			},
		},
		{
			name: "query specific service",
			args: args{
				log:      logger,
				basePath: "/",
				h:        []healthstatus.HealthCheck{&succeedingCheck{}, &failingCheck{}},
				service:  "success",
			},
			want: &HealthResponse{
				Status:  healthstatus.HealthStatusHealthy,
				Message: "",
				Services: map[string]HealthResponse{
					"success": {
						Status:  healthstatus.HealthStatusHealthy,
						Message: "",
						Services: map[string]HealthResponse{
							"successPartition": {
								Status:  healthstatus.HealthStatusHealthy,
								Message: "",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ws, err := NewHealthGroup(tt.args.log, tt.args.basePath, tt.args.h...)
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
