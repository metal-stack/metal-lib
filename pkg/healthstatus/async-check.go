package healthstatus

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type AsyncHealthCheck struct {
	healthCheck         HealthCheck
	log                 *slog.Logger
	healthCheckInterval time.Duration

	lock    sync.RWMutex
	current currentState
	ticker  *time.Ticker
}

func Async(log *slog.Logger, interval time.Duration, hc HealthCheck) *AsyncHealthCheck {
	return &AsyncHealthCheck{
		healthCheckInterval: interval,
		healthCheck:         hc,
		log:                 log.With("type", "async", "service", hc.ServiceName()),
		current: currentState{
			status: HealthResult{
				Status:  HealthStatusHealthy,
				Message: "",
			},
		},
	}
}

func (c *AsyncHealthCheck) ServiceName() string {
	return c.healthCheck.ServiceName()
}

func (c *AsyncHealthCheck) Check(_ context.Context) (HealthResult, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.current.status, c.current.err
}

func (r *AsyncHealthCheck) Start(ctx context.Context) {
	r.log.Debug("started async updates")
	if r.ticker != nil {
		r.ticker.Reset(r.healthCheckInterval)
	} else {
		r.ticker = time.NewTicker(r.healthCheckInterval)
	}
	go func() {
		r.lock.Lock()
		err := r.updateStatus(ctx)
		if err != nil {
			r.log.Error("async services are unhealthy", "error", err)
		}
		r.lock.Unlock()

		for {
			select {
			case <-ctx.Done():
				r.log.Info("stop async health checking, context is done")
				r.Stop(ctx)
				return
			case <-r.ticker.C:
				if r.lock.TryLock() {
					err := r.updateStatus(ctx)
					if err != nil {
						r.log.Error("services are unhealthy", "error", err)
					}
					r.lock.Unlock()
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
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.updateStatus(ctx)
	if err != nil {
		r.log.Error("services are unhealthy", "error", err)
	}
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
