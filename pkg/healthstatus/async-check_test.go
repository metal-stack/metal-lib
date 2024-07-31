package healthstatus

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type countedCheck struct {
	name   string
	state  currentState
	checks int
}

func (c *countedCheck) ServiceName() string {
	return c.name
}
func (c *countedCheck) Check(context.Context) (HealthResult, error) {
	c.checks++
	return c.state.status, c.state.err
}

func TestAsync(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	log := slog.Default()

	tests := []struct {
		name          string
		interval      time.Duration
		callIntervals []time.Duration
		wantChecks    int
		hc            *countedCheck
		want          currentState
	}{
		{
			name:     "succeeding call",
			interval: 2 * time.Second,
			callIntervals: []time.Duration{
				500 * time.Millisecond,
				100 * time.Millisecond,
			},
			wantChecks: 1,
			hc: &countedCheck{
				name: "succeeding call",
				state: currentState{
					status: HealthResult{
						Status: HealthStatusHealthy,
					},
				},
			},
			want: currentState{
				status: HealthResult{
					Status: HealthStatusHealthy,
				},
			},
		},
		// {
		// 	name:     "multiple calls",
		// 	interval: 2 * time.Second,
		// 	callIntervals: []time.Duration{
		// 		500 * time.Millisecond,
		// 		600 * time.Millisecond,
		// 	},
		// 	wantChecks: 1,
		// 	hc: &countedCheck{
		// 		name: "multiple calls",
		// 		state: currentState{
		// 			status: HealthResult{
		// 				Status: HealthStatusHealthy,
		// 			},
		// 		},
		// 	},
		// 	want: currentState{
		// 		status: HealthResult{
		// 			Status: HealthStatusHealthy,
		// 		},
		// 	},
		// },
		// {
		// 	name:     "error propagated",
		// 	interval: 2 * time.Second,
		// 	callIntervals: []time.Duration{
		// 		100 * time.Millisecond,
		// 	},
		// 	wantChecks: 1,
		// 	hc: &countedCheck{
		// 		name: "error propagated",
		// 		state: currentState{
		// 			status: HealthResult{
		// 				Status: HealthStatusUnhealthy,
		// 			},
		// 			err: errors.New("intentional"),
		// 		},
		// 	},
		// 	want: currentState{
		// 		status: HealthResult{
		// 			Status: HealthStatusUnhealthy,
		// 		},
		// 		err: errors.New("intentional"),
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			hc := Async(log, tt.interval, tt.hc)
			hc.Start(ctx)

			for _, timeout := range tt.callIntervals {
				time.Sleep(timeout)

				got, gotErr := hc.Check(ctx)

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

			}

			if tt.hc.checks != tt.wantChecks {
				t.Errorf("mismatched calls (want %d, got %d)", tt.wantChecks, tt.hc.checks)
			}
		})
	}
}
