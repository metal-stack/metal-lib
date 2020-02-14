package rest

import (
	"net/http"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"go.uber.org/zap"
)

// HealthCheck is a normal function which returns an error if something
// is not correct.
type HealthCheck func() error

// HealthStatus indicates the health of a webservice.
type HealthStatus string

const (
	// HealthStatusHealthy is returned when the service is healthy
	HealthStatusHealthy = "healthy"
	// HealthStatusUnhealthy is returned when the service is not healthy
	HealthStatusUnhealthy = "unhealthy"
)

type status struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message"`
}

func emptyCheck() error {
	return nil
}

// NewHealth returns a webservice for healthchecks. All checks are
// executed until one returns an error or all of them succeed. If
// no healthcheck is given, a default check will be executed.
func NewHealth(log *zap.Logger, basePath string, h ...HealthCheck) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(basePath + "v1/health").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"health"}
	if h == nil {
		h = []HealthCheck{emptyCheck}
	}

	ws.Route(ws.GET("/").To(check(log, h...)).
		Operation("health").
		Doc("perform a healthcheck").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", status{}).
		Returns(http.StatusInternalServerError, "Unhealthy", status{}))
	return ws
}

func check(log *zap.Logger, handlers ...HealthCheck) func(request *restful.Request, response *restful.Response) {
	return func(request *restful.Request, response *restful.Response) {
		for _, h := range handlers {
			if h == nil {
				continue
			}
			e := h()
			if e != nil {
				s := status{
					Status:  HealthStatusUnhealthy,
					Message: e.Error(),
				}
				log.Error("unhealthy", zap.Error(e))
				err := response.WriteHeaderAndEntity(http.StatusInternalServerError, s)
				if err != nil {
					log.Error("writeHeaderAndEntity", zap.Error(err))
				}
				return
			}
		}
		s := status{
			Status:  HealthStatusHealthy,
			Message: "OK",
		}
		err := response.WriteEntity(s)
		if err != nil {
			log.Error("writeEntity", zap.Error(err))
		}
	}
}
