package healthstatus

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/semaphore"
)

type AsyncHealthCheck struct {
	healthCheck         HealthCheck
	log                 *slog.Logger
	healthCheckInterval time.Duration

	sem     *semaphore.Weighted
	current currentState
	ticker  *time.Ticker
}

func Async(log *slog.Logger, interval time.Duration, hc HealthCheck) *AsyncHealthCheck {
	return &AsyncHealthCheck{
		healthCheckInterval: interval,
		healthCheck:         hc,
		log:                 log,
		sem:                 semaphore.NewWeighted(1),
		current: currentState{
			Status: HealthResult{
				Status:  HealthStatusHealthy,
				Message: "",
			},
		},
	}
}

func (c *AsyncHealthCheck) ServiceName() string {
	return c.healthCheck.ServiceName()
}

func (c *AsyncHealthCheck) Check(context.Context) (HealthResult, error) {
	c.log.Debug("checked async")
	if c.ticker == nil {
		// The context coming in is bound to a single request
		// but the ticker should be started in background
		c.Start(context.Background()) //nolint:contextcheck
	}
	return c.current.Status, c.current.Err
}

func (r *AsyncHealthCheck) Start(ctx context.Context) {
	r.log.Debug("started async updates")
	if r.ticker != nil {
		r.ticker.Reset(r.healthCheckInterval)
	} else {
		r.ticker = time.NewTicker(r.healthCheckInterval)
	}
	go func() {
		err := r.updateStatus(ctx)
		if err != nil {
			r.log.Error("services are unhealthy", "error", err)
		}

		for {
			select {
			case <-ctx.Done():
				r.log.Info("stop health checking, context is done")
				return
			case <-r.ticker.C:
				if r.sem.TryAcquire(1) {
					err := r.updateStatus(ctx)
					if err != nil {
						r.log.Error("services are unhealthy", "error", err)
					}
					r.sem.Release(1)
				} else {
					r.log.Info("skip updating health status because update is still running")
				}
			}
		}
	}()
}

func (r *AsyncHealthCheck) Stop(ctx context.Context) {
	r.ticker.Stop()
}

func (r *AsyncHealthCheck) ForceUpdateStatus(ctx context.Context) error {
	err := r.sem.Acquire(ctx, 1)
	if err != nil {
		return err
	}
	err = r.updateStatus(ctx)
	if err != nil {
		r.log.Error("services are unhealthy", "error", err)
	}
	r.sem.Release(1)
	return err
}

func (r *AsyncHealthCheck) updateStatus(ctx context.Context) error {
	r.log.Info("evaluating current service health statuses")
	ctx, cancel := context.WithTimeout(ctx, r.healthCheckInterval/2)
	defer cancel()

	res, err := r.healthCheck.Check(ctx)
	r.current = currentState{res, err}
	r.log.Debug("evaluated current service health statuses", "current", r.current)
	return err
}
