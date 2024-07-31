package healthstatus

import "context"

// HealthStatus indicates the health of a service.
type HealthStatus string

const (
	// HealthStatusHealthy is returned when the service is healthy.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy is returned when the service is not healthy.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded is returned when the service is degraded.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusPartiallyUnhealthy is returned when the service is partially not healthy.
	HealthStatusPartiallyUnhealthy HealthStatus = "partial-outage"
)

// HealthCheck defines an interface for health checks.
type HealthCheck interface {
	// ServiceName returns the name of the service that is health checked.
	ServiceName() string
	// Check is a function returning a service status and an error.
	Check(ctx context.Context) (HealthResult, error)
}

// HealthResult holds the health state of a service.
type HealthResult struct {
	// Status indicates the overall health state.
	Status HealthStatus
	// Message gives additional information on the overall health state.
	Message string
	// Services contain the individual health results of the services as evaluated by the HealthCheck interface. The overall HealthStatus is then derived automatically from the results of the health checks.
	//
	// Note that the individual HealthResults evaluated by the HealthCheck interface may again consist of a plurality services. While this is only optional it allows for creating nested health structures. These can be used for more sophisticated scenarios like evaluating platform health describing service availability in different locations or similar.
	//
	// If using nested HealthResults, the status of the parent service can be derived automatically from the status of its children by leaving the parent's health status field blank.
	Services map[string]HealthResult
}

type currentState struct {
	status HealthResult
	err    error
}
