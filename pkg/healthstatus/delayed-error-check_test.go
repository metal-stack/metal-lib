package healthstatus

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type recordedCheck struct {
	name   string
	states []currentState
}

func (c *recordedCheck) ServiceName() string {
	return c.name
}
func (c *recordedCheck) Check(context.Context) (HealthResult, error) {
	cur := c.states[0]
	if len(c.states) > 1 {
		c.states = c.states[1:]
	}
	return cur.status, cur.err
}

func TestDelayErrors(t *testing.T) {
	log := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	tests := []struct {
		name string
		hc   *DelayedErrorHealthCheck
		want []currentState
	}{
		{
			name: "check always returns first result even on error",
			hc: DelayErrors(log, 1, &recordedCheck{
				name: "record",
				states: []currentState{
					{
						status: HealthResult{
							Status:   HealthStatusUnhealthy,
							Message:  "intentional",
							Services: map[string]HealthResult{},
						},
						err: errors.New("initial error"),
					},
					{
						status: HealthResult{
							Status:   HealthStatusUnhealthy,
							Message:  "intentional",
							Services: map[string]HealthResult{},
						},
						err: errors.New("secondary error"),
					},
				},
			}),
			want: []currentState{
				{
					status: HealthResult{
						Status:   HealthStatusUnhealthy,
						Message:  "intentional",
						Services: map[string]HealthResult{},
					},
					err: errors.New("initial error"),
				},
				{
					status: HealthResult{
						Status:   HealthStatusUnhealthy,
						Message:  "intentional",
						Services: map[string]HealthResult{},
					},
					err: errors.New("secondary error"),
				},
			},
		},
		{
			name: "ignores first error after initial success",
			hc: DelayErrors(log, 1, &recordedCheck{
				name: "record",
				states: []currentState{
					{
						status: HealthResult{
							Status:   HealthStatusHealthy,
							Message:  "",
							Services: map[string]HealthResult{},
						},
					},
					{
						status: HealthResult{
							Status:   HealthStatusUnhealthy,
							Message:  "intentional",
							Services: map[string]HealthResult{},
						},
						err: errors.New("hidden error"),
					},
					{
						status: HealthResult{
							Status:   HealthStatusUnhealthy,
							Message:  "intentional",
							Services: map[string]HealthResult{},
						},
						err: errors.New("presented error"),
					},
				},
			}),
			want: []currentState{
				{
					status: HealthResult{
						Status:   HealthStatusHealthy,
						Message:  "",
						Services: map[string]HealthResult{},
					},
				},
				{
					status: HealthResult{
						Status:   HealthStatusHealthy,
						Message:  "",
						Services: map[string]HealthResult{},
					},
				},
				{
					status: HealthResult{
						Status:   HealthStatusUnhealthy,
						Message:  "intentional",
						Services: map[string]HealthResult{},
					},
					err: errors.New("presented error"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			for i, w := range tt.want {
				got, gotErr := tt.hc.Check(ctx)

				var (
					wantErrStr = "<nil>"
					gotErrStr  = "<nil>"
				)
				if w.err != nil {
					wantErrStr = w.err.Error()
				}
				if gotErr != nil {
					gotErrStr = gotErr.Error()
				}
				if diff := cmp.Diff(wantErrStr, gotErrStr); diff != "" {
					t.Errorf("mismatch error on call %d (-want +got):\n%s", i, diff)
				}
				if diff := cmp.Diff(w.status, got); diff != "" {
					t.Errorf("mismatch result on call %d (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}
