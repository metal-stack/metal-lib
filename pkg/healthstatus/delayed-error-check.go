package healthstatus

import (
	"context"
)

type DelayedErrorHealthCheck struct {
	maxIgnoredErrors       int
	errorCountSinceSuccess int
	lastSuccess            currentState
	healthCheck            HealthCheck
}

func DelayErrors(maxIgnoredErrors int, hc HealthCheck) *DelayedErrorHealthCheck {
	return &DelayedErrorHealthCheck{
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
	return c.lastSuccess.status, c.lastSuccess.err
}
