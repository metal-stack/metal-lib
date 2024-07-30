package healthstatus

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type staticCheck struct {
	name  string
	state currentState
}

func (c *staticCheck) ServiceName() string {
	return c.name
}
func (c *staticCheck) Check(context.Context) (HealthResult, error) {
	return c.state.status, c.state.err
}
func TestGrouped(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	log := slog.Default()

	tests := []struct {
		name string
		hcs  []HealthCheck
		want currentState
	}{
		{
			name: "adds subchecks as service by name",
			hcs: []HealthCheck{
				&staticCheck{
					name: "a",
					state: currentState{
						status: HealthResult{
							Status:   HealthStatusHealthy,
							Message:  "",
							Services: map[string]HealthResult{},
						},
					},
				},
				&staticCheck{
					name: "b",
					state: currentState{
						status: HealthResult{
							Status:   HealthStatusDegraded,
							Message:  "bees are tired",
							Services: map[string]HealthResult{},
						},
					},
				},
			},
			want: currentState{
				status: HealthResult{
					Status:  HealthStatusDegraded,
					Message: "",
					Services: map[string]HealthResult{
						"a": {
							Status:   HealthStatusHealthy,
							Message:  "",
							Services: map[string]HealthResult{},
						},
						"b": {
							Status:   HealthStatusDegraded,
							Message:  "bees are tired",
							Services: map[string]HealthResult{},
						},
					},
				},
			},
		},
		{
			name: "errors if one errors",
			hcs: []HealthCheck{
				&staticCheck{
					name: "a",
					state: currentState{
						status: HealthResult{
							Status:   HealthStatusUnhealthy,
							Message:  "",
							Services: map[string]HealthResult{},
						},
						err: errors.New("intentional error"),
					},
				},
				&staticCheck{
					name: "b",
					state: currentState{
						status: HealthResult{
							Status:   HealthStatusDegraded,
							Message:  "bees are tired",
							Services: map[string]HealthResult{},
						},
					},
				},
			},
			want: currentState{
				status: HealthResult{
					Status:  HealthStatusPartiallyUnhealthy,
					Message: "intentional error",
					Services: map[string]HealthResult{
						"a": {
							Status:   HealthStatusUnhealthy,
							Message:  "intentional error",
							Services: map[string]HealthResult{},
						},
						"b": {
							Status:   HealthStatusDegraded,
							Message:  "bees are tired",
							Services: map[string]HealthResult{},
						},
					},
				},
				err: errors.New("intentional error"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			group := Grouped(log, tt.name, tt.hcs...)

			got, gotErr := group.Check(ctx)

			var (
				wantErrStr = "<nil>"
				gotErrStr  = "<nil>"
			)
			if tt.want.err != nil {
				wantErrStr = tt.want.err.Error()
			}
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if diff := cmp.Diff(wantErrStr, gotErrStr); diff != "" {
				t.Errorf("mismatch error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.status, got); diff != "" {
				t.Errorf("mismatch result (-want +got):\n%s", diff)
			}
		})
	}
}
