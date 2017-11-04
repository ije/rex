package webx

import (
	"strconv"
	"strings"

	"github.com/ije/webx/acl"
)

type APIService struct {
	Prefix string
	route  map[string]map[string]*apiHandler
}

type apiHandler struct {
	handle    APIHandle
	privilege *acl.Privilege
}

type APIHandle func(*Context, *XService)

// Options
// apis.Options('_private_endpoint', &webx.CORS{Origin: "*"})
func (s APIService) Options(endpoint string, cors *CORS) {
	if cors == nil {
		return
	}

	s.register("OPTIONS", endpoint, func(ctx *Context, xs *XService) {
		w := ctx.ResponseWriter()
		wh := ctx.ResponseWriter().Header()
		wh.Set("Access-Control-Allow-Origin", cors.Origin)
		wh.Set("Access-Control-Allow-Methods", strings.Join(cors.Methods, ","))
		wh.Set("Access-Control-Allow-Headers", strings.Join(cors.Headers, ","))
		if cors.Credentials {
			wh.Set("Access-Control-Allow-Credentials", "true")
		}
		if cors.MaxAge > 0 {
			wh.Set("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
		}
		w.WriteHeader(204)
	}, "")
}

func (s APIService) Head(endpoint string, handle APIHandle, privilegeId string) {
	s.register("HEAD", endpoint, handle, privilegeId)
}

func (s APIService) Get(endpoint string, handle APIHandle, privilegeId string) {
	s.register("GET", endpoint, handle, privilegeId)
}

func (s APIService) Post(endpoint string, handle APIHandle, privilegeId string) {
	s.register("POST", endpoint, handle, privilegeId)
}

func (s APIService) Put(endpoint string, handle APIHandle, privilegeId string) {
	s.register("PUT", endpoint, handle, privilegeId)
}

func (s APIService) Patch(endpoint string, handle APIHandle, privilegeId string) {
	s.register("PATCH", endpoint, handle, privilegeId)
}

func (s APIService) Delete(endpoint string, handle APIHandle, privilegeId string) {
	s.register("DELETE", endpoint, handle, privilegeId)
}

func (s APIService) register(method string, endpoint string, handle APIHandle, privilegeId string) {
	if handle == nil {
		return
	}

	if len(endpoint) == 0 {
		return
	}

	var privilege *acl.Privilege = nil
	if len(privilegeId) > 0 {
		privilege = acl.NewPrivilege(privilegeId)
	}

	if s.route[method] == nil {
		s.route[method] = map[string]*apiHandler{}
	}
	s.route[method][endpoint] = &apiHandler{privilege: privilege, handle: handle}
}

func Register(apis *APIService) {
	apisMux.Register(apis)
}
