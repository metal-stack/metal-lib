package healthstatus

import (
	"context"
)

type DeferredErrorHealthCheck struct {
	maxIgnoredErrors       int
	errorCountSinceSuccess int
	lastSuccess            currentState
	healthCheck            HealthCheck
}

func DeferErrors(maxIgnoredErrors int, hc HealthCheck) *DeferredErrorHealthCheck {
	return &DeferredErrorHealthCheck{
		maxIgnoredErrors: maxIgnoredErrors,
		healthCheck:      hc,
		lastSuccess: currentState{
			Status: HealthResult{
				Status:  HealthStatusHealthy,
				Message: "",
			},
		},
	}
}

func (c *DeferredErrorHealthCheck) ServiceName() string {
	return c.healthCheck.ServiceName()
}

func (c *DeferredErrorHealthCheck) Check(ctx context.Context) (HealthResult, error) {
	status, err := c.healthCheck.Check(ctx)
	state := currentState{status, err}

	if err == nil {
		c.errorCountSinceSuccess = 0
		c.lastSuccess = state
		return status, err
	}
	c.errorCountSinceSuccess++
	if c.errorCountSinceSuccess > c.maxIgnoredErrors {
		return status, err
	}
	return c.lastSuccess.Status, c.lastSuccess.Err
}
