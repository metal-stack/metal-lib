package rest

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/v"
)

// name this struct "version", so go-swagger will generate a type named "version"
type version struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	BuildDate string `json:"builddate"`
	Revision  string `json:"revision"`
	Gitsha1   string `json:"gitsha1"`
}

// NewVersion returns a webservice which returns version information. The given
// name should be a descriptive name of the module.
func NewVersion(name string, basePath string) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(basePath + "v1/version").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"version"}

	vi := version{
		Name:      name,
		Version:   v.Version,
		Revision:  v.Revision,
		BuildDate: v.BuildDate,
		Gitsha1:   v.GitSHA1,
	}
	ws.Route(
		ws.GET("/").
			Doc("returns the current version information of this module").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			Returns(http.StatusOK, "OK", version{}).
			Operation("info").
			To(func(r *restful.Request, rsp *restful.Response) {
				_ = rsp.WriteAsJson(vi)
			}))

	return ws
}
