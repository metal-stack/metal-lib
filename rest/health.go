package rest

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

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

// HealthResponse is returned by the API when executing a health check.
type HealthResponse struct {
	// Status indicates the overall health state.
	Status HealthStatus `json:"status"`
	// Message gives additional information on the overall health state.
	Message string `json:"message"`
	// Services contain the individual health results of the services as evaluated by the HealthCheck interface. The overall HealthStatus is then derived automatically from the results of the health checks.
	//
	// Note that the individual HealthResults evaluated by the HealthCheck interface may again consist of a plurality services. While this is only optional it allows for creating nested health structures. These can be used for more sophisticated scenarios like evaluating platform health describing service availability in different locations or similar.
	//
	// If using nested HealthResults, the status of the parent service can be derived automatically from the status of its children by leaving the parent's health status field blank.
	Services map[string]HealthResult `json:"services"`
}

// HealthResult holds the health state of a service.
type HealthResult HealthResponse

type healthResource struct {
	log          *zap.SugaredLogger
	healthChecks map[string]HealthCheck
}

// NewHealth returns a webservice for healthchecks. All checks are
// executed and returned in a service health map.
func NewHealth(log *zap.Logger, basePath string, healthChecks ...HealthCheck) (*restful.WebService, error) {
	h := &healthResource{
		log:          log.Sugar(),
		healthChecks: map[string]HealthCheck{},
	}

	for _, healthCheck := range healthChecks {
		name := healthCheck.ServiceName()
		if name == "" {
			return nil, fmt.Errorf("health check service name should not be empty")
		}
		_, ok := h.healthChecks[name]
		if ok {
			return nil, fmt.Errorf("health checks must register with unique names")
		}
		h.healthChecks[name] = healthCheck
	}

	return h.webService(basePath), nil
}

func (h *healthResource) webService(basePath string) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(basePath + "v1/health").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"health"}

	ws.Route(ws.GET("/").To(h.check).
		Operation("health").
		Doc("perform a healthcheck").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("service", "return health for this specific service only").DataType("string")).
		Returns(http.StatusOK, "OK", HealthResponse{}).
		Returns(http.StatusInternalServerError, "Unhealthy", HealthResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (h *healthResource) check(request *restful.Request, response *restful.Response) {
	type chanResult struct {
		name string
		HealthResult
	}

	var (
		service = request.QueryParameter("service")
		result  = HealthResponse{
			Status:   HealthStatusHealthy,
			Message:  "",
			Services: map[string]HealthResult{},
		}

		resultChan = make(chan chanResult)
		once       sync.Once
	)
	defer once.Do(func() { close(resultChan) })

	ctx := request.Request.Context()
	g, _ := errgroup.WithContext(ctx)

	for name, healthCheck := range h.healthChecks {
		name := name
		healthCheck := healthCheck

		g.Go(func() error {
			if h == nil {
				return nil
			}
			if service != "" && name != service {
				return nil
			}

			result := chanResult{
				name: name,
				HealthResult: HealthResult{
					Status:   HealthStatusHealthy,
					Message:  "",
					Services: map[string]HealthResult{},
				},
			}
			defer func() {
				resultChan <- result
			}()

			var err error
			result.HealthResult, err = healthCheck.Check(ctx)
			if err != nil {
				result.Message = err.Error()
				h.log.Errorw("unhealthy service", "name", name, "status", result.Status, "error", err)
			}

			return err
		})
	}

	finished := make(chan bool)
	go func() {
		for r := range resultChan {
			r := r
			result.Services[r.name] = r.HealthResult
		}
		finished <- true
	}()

	rc := http.StatusOK

	if err := g.Wait(); err != nil {
		rc = http.StatusInternalServerError
		result.Message = err.Error()
	}

	once.Do(func() { close(resultChan) })

	<-finished

	result.Status = DeriveOverallHealthStatus(result.Services)

	err := response.WriteHeaderAndEntity(rc, result)
	if err != nil {
		h.log.Error("error writing response", zap.Error(err))
	}
}

func DeriveOverallHealthStatus(information map[string]HealthResult) HealthStatus {
	var (
		result    = HealthStatusHealthy
		degraded  int
		unhealthy int
	)

	for _, service := range information {
		if service.Status == "" {
			service.Status = DeriveOverallHealthStatus(service.Services)
		}
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

	if len(information) > 0 {
		if degraded > 0 {
			result = HealthStatusDegraded
		}
		if unhealthy > 0 {
			result = HealthStatusPartiallyUnhealthy
		}
		if unhealthy == len(information) {
			result = HealthStatusUnhealthy
		}
	}

	return result
}
