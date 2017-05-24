package webx

import (
	"strings"

	"github.com/ije/webx/user"
)

var xapis []APIService

type apiHandler struct {
	privileges user.Privileges
	handle     interface{}
}

type APIService map[string]map[string]apiHandler

// apis.Config('prefix', 'v2')
// apis.Config('prefix', '_private')
func (s APIService) Config(key string, value string) {
	if s["CONFIG"] == nil {
		s["CONFIG"] = map[string]apiHandler{}
	}
	s["CONFIG"][key] = apiHandler{handle: func() string { return value }}
}

func (s APIService) getConfig(key string) (value string) {
	if config, yes := s["CONFIG"]; yes {
		if h, yes := config[key]; yes {
			if f, yes := h.handle.(func() string); yes {
				value = f()
			}
		}
	}
	return
}

// apis.Options('*', PublicCORS())
// apis.Options('_private/endpoint', nil)
func (s APIService) Options(endpoint string, cors *CORS) {
	if s["OPTIONS"] == nil {
		s["OPTIONS"] = map[string]apiHandler{}
	}
	s["OPTIONS"][strings.Trim(endpoint, "/")] = apiHandler{handle: func() *CORS { return cors }}
}

func (s APIService) Head(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("HEAD", endpoint, handle, privileges)
}

func (s APIService) Get(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("GET", endpoint, handle, privileges)
}

func (s APIService) Post(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("POST", endpoint, handle, privileges)
}

func (s APIService) Put(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("PUT", endpoint, handle, privileges)
}

func (s APIService) Patch(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("PATCH", endpoint, handle, privileges)
}

func (s APIService) Delete(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("DELETE", endpoint, handle, privileges)
}

func (s APIService) register(method string, endpoint string, handle interface{}, privileges user.Privileges) {
	switch v := handle.(type) {
	case func(), func(*Context), func(*XService), func(*Context, *XService), func(*XService, *Context):
		if s[method] == nil {
			s[method] = map[string]apiHandler{}
		}
		s[method][strings.Trim(endpoint, "/")] = apiHandler{privileges, v}
	default:
		panic("register %s /api/%s: the handle should be a valid api handle\n available api handle types:\n\tfunc()\n\tfunc(*webx.Context)\n\tfunc(*webx.XService)`, 1func(*webx.Context, *webx.XService)\n\tfunc(*webx.XService, *webx.Context)")
	}
}

func Register(apis APIService) {
	xapis = append(xapis, apis)
}
