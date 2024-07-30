package rest

import (
	"log/slog"
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
)

// HealthResponse is returned by the API when executing a health check.
type HealthResponse struct {
	// Status indicates the overall health state.
	Status healthstatus.HealthStatus `json:"status"`
	// Message gives additional information on the overall health state.
	Message string `json:"message"`
	// Services contain the individual health results of the services as evaluated by the HealthCheck interface. The overall HealthStatus is then derived automatically from the results of the health checks.
	//
	// Note that the individual HealthResults evaluated by the HealthCheck interface may again consist of a plurality services. While this is only optional it allows for creating nested health structures. These can be used for more sophisticated scenarios like evaluating platform health describing service availability in different locations or similar.
	//
	// If using nested HealthResults, the status of the parent service can be derived automatically from the status of its children by leaving the parent's health status field blank.
	Services map[string]HealthResponse `json:"services,omitempty"`
}

type healthResource struct {
	log         *slog.Logger
	healthCheck healthstatus.HealthCheck
}

// NewHealth returns a webservice for healthchecks. All checks are
// executed and returned in a service health map.
func NewHealth(log *slog.Logger, basePath string, healthChecks ...healthstatus.HealthCheck) (*restful.WebService, error) {
	h := &healthResource{
		log: log,
	}
	if len(healthChecks) == 1 {
		h.healthCheck = healthChecks[0]
	} else {
		h.healthCheck = healthstatus.Grouped(log, "", healthChecks...)
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
	var (
		service = request.QueryParameter("service")
		ctx     = request.Request.Context()
		code    = http.StatusOK
	)

	result, err := h.healthCheck.Check(ctx)
	if err != nil {
		h.log.Error("unhealthy application", "status", result.Status, "error", err)

		code = http.StatusInternalServerError
		if result.Status == "" {
			result.Status = healthstatus.DeriveOverallHealthStatus(result.Services)
		}
		if result.Message == "" {
			result.Message = err.Error()
		}
	}

	hr := resultToResponse(result)
	if service != "" && hr.Services != nil {
		srv := hr.Services[service]
		hr.Status = srv.Status
		hr.Message = srv.Message
		hr.Services = map[string]HealthResponse{service: srv}
	}

	err = response.WriteHeaderAndEntity(code, hr)
	if err != nil {
		h.log.Error("error writing response", "error", err)
	}
}

func resultToResponse(result healthstatus.HealthResult) HealthResponse {
	hr := HealthResponse{
		Status:  result.Status,
		Message: result.Message,
	}
	if result.Services != nil {
		hr.Services = make(map[string]HealthResponse)
	}
	for name, serviceResult := range result.Services {
		hr.Services[name] = resultToResponse(serviceResult)
	}
	return hr
}
