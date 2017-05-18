package webx

import (
	"strings"

	"github.com/ije/webx/user"
)

var xapis = APIService{}

type apiHandler struct {
	privileges user.Privileges
	handle     interface{}
}

type APIService map[string]map[string]apiHandler

func (s APIService) Get(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("GET", endpoint, handle, privileges)
}

func (s APIService) Post(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("POST", endpoint, handle, privileges)
}

func (s APIService) Put(endpoint string, handle interface{}, privileges user.Privileges) {
	s.register("PUT", endpoint, handle, privileges)
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
		endpoint = strings.Trim(endpoint, "/")
		s[method][endpoint] = apiHandler{privileges, v}
	default:
		panic("register %s /api/%s: the handle should be a valid api handle\n available api handle types:\n\tfunc()\n\tfunc(*webx.Context)\n\tfunc(*webx.XService)`, 1func(*webx.Context, *webx.XService)\n\tfunc(*webx.XService, *webx.Context)")
	}
}

func Register(apis APIService) {
	for method, handlers := range apis {
		for endpoint, handler := range handlers {
			xapis.register(method, endpoint, handler.handle, handler.privileges)
		}
	}
}
