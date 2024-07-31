package healthstatus

import (
	"context"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"
)

type groupedHealthCheck struct {
	serviceName string
	hcs         []HealthCheck
	log         *slog.Logger
}

func Grouped(log *slog.Logger, serviceName string, checks ...HealthCheck) *groupedHealthCheck {
	return &groupedHealthCheck{
		serviceName: serviceName,
		hcs:         checks,
		log:         log.With("group", serviceName, "type", "group"),
	}
}

func (c *groupedHealthCheck) Add(hc HealthCheck) {
	c.hcs = append(c.hcs, hc)
}

func (c *groupedHealthCheck) ServiceName() string {
	return c.serviceName
}
func (c *groupedHealthCheck) Check(ctx context.Context) (HealthResult, error) {
	type chanResult struct {
		name string
		HealthResult
	}
	if len(c.hcs) == 0 {
		return HealthResult{
			Status:   HealthStatusHealthy,
			Message:  "",
			Services: nil,
		}, nil
	}
	var (
		result = HealthResult{
			Status:   HealthStatusHealthy,
			Message:  "",
			Services: map[string]HealthResult{},
		}

		resultChan = make(chan chanResult)
		once       sync.Once
	)
	defer once.Do(func() { close(resultChan) })

	g, _ := errgroup.WithContext(ctx)

	for _, healthCheck := range c.hcs {
		name := healthCheck.ServiceName()
		healthCheck := healthCheck

		g.Go(func() error {
			result := chanResult{
				name: name,
				HealthResult: HealthResult{
					Status:  HealthStatusHealthy,
					Message: "",
				},
			}
			defer func() {
				resultChan <- result
			}()

			var err error
			result.HealthResult, err = healthCheck.Check(ctx)
			if err != nil {
				result.Message = err.Error()
				c.log.Error("unhealthy service", "name", name, "status", result.Status, "error", err)
			}

			return err
		})
	}

	finished := make(chan bool)
	go func() {
		for r := range resultChan {
			result.Services[r.name] = r.HealthResult
		}
		finished <- true
	}()
	err := g.Wait()
	once.Do(func() { close(resultChan) })

	<-finished

	if err != nil {
		result.Message = err.Error()
		result.Status = HealthStatusUnhealthy
	}
	result.Status = DeriveOverallHealthStatus(result.Services)
	return result, err
}

func DeriveOverallHealthStatus(services map[string]HealthResult) HealthStatus {
	var (
		result    = HealthStatusHealthy
		degraded  int
		unhealthy int
	)

	for k, service := range services {
		if len(service.Services) > 0 && service.Status == "" {
			service.Status = DeriveOverallHealthStatus(service.Services)
		}
		services[k] = service
		switch service.Status {
		case HealthStatusHealthy:
		case HealthStatusDegraded:
			degraded++
		case HealthStatusUnhealthy, HealthStatusPartiallyUnhealthy:
			unhealthy++
		default:
			unhealthy++
		}
	}

	if len(services) > 0 {
		if degraded > 0 {
			result = HealthStatusDegraded
		}
		if unhealthy > 0 {
			result = HealthStatusPartiallyUnhealthy
		}
		if unhealthy == len(services) {
			result = HealthStatusUnhealthy
		}
	}

	return result
}
