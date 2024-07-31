package healthstatus

import (
	"context"
	"log/slog"
)

type DelayedErrorHealthCheck struct {
	maxIgnoredErrors       int
	errorCountSinceSuccess int
	lastSuccess            currentState
	healthCheck            HealthCheck
	log                    *slog.Logger
}

func DelayErrors(log *slog.Logger, maxIgnoredErrors int, hc HealthCheck) *DelayedErrorHealthCheck {
	return &DelayedErrorHealthCheck{
		log:              log.With("type", "delay"),
		maxIgnoredErrors: maxIgnoredErrors,
		healthCheck:      hc,
		// trick the check to always start with the actual state
		errorCountSinceSuccess: maxIgnoredErrors,
	}
}

func (c *DelayedErrorHealthCheck) ServiceName() string {
	return c.healthCheck.ServiceName()
}

func (c *DelayedErrorHealthCheck) Check(ctx context.Context) (HealthResult, error) {
	status, err := c.healthCheck.Check(ctx)
	state := currentState{status, err}

	if err == nil {
		c.errorCountSinceSuccess = 0
		c.lastSuccess = state
		return status, nil
	}
	c.errorCountSinceSuccess++
	if c.errorCountSinceSuccess > c.maxIgnoredErrors {
		return status, err
	}
	c.log.Warn("delaying health check error propagation", "counter", c.errorCountSinceSuccess, "max", c.maxIgnoredErrors, "err", err, "status", status.Status)
	return c.lastSuccess.status, c.lastSuccess.err
}
