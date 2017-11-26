package webx

import (
	"strconv"
	"strings"

	"github.com/ije/webx/acl"
)

type APIService struct {
	Prefix      string
	middlewares []APIHandle
	route       map[string]map[string]*apiHandler
}

type apiHandler struct {
	handle     APIHandle
	privileges map[string]acl.Privilege
}

type APIHandle func(*Context, *XService)

func (s *APIService) Use(middleware APIHandle) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *APIService) OPTIONS(endpoint string, cors *CORS) {
	if cors == nil {
		return
	}

	s.register("OPTIONS", endpoint, func(ctx *Context, xs *XService) {
		w := ctx.ResponseWriter
		wh := w.Header()

		if len(cors.Origin) > 0 {
			wh.Set("Access-Control-Allow-Origin", cors.Origin)
			wh.Set("Vary", "Origin")

			if len(cors.Methods) > 0 {
				wh.Set("Access-Control-Allow-Methods", strings.Join(cors.Methods, ", "))
			}
			if len(cors.Headers) > 0 {
				wh.Set("Access-Control-Allow-Headers", strings.Join(cors.Headers, ", "))
			}
			if cors.Credentials {
				wh.Set("Access-Control-Allow-Credentials", "true")
			}
			if cors.MaxAge > 0 {
				wh.Set("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
			}
		}

		w.WriteHeader(204)
	}, nil)
}

func (s *APIService) HEAD(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("HEAD", endpoint, handle, privilegeIds)
}

func (s *APIService) GET(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("GET", endpoint, handle, privilegeIds)
}

func (s *APIService) POST(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("POST", endpoint, handle, privilegeIds)
}

func (s *APIService) PUT(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("PUT", endpoint, handle, privilegeIds)
}

func (s *APIService) PATCH(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("PATCH", endpoint, handle, privilegeIds)
}

func (s *APIService) DELETE(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("DELETE", endpoint, handle, privilegeIds)
}

func (s *APIService) register(method string, endpoint string, handle APIHandle, privilegeIds []string) {
	if handle == nil {
		return
	}

	if len(endpoint) == 0 {
		return
	}

	var privileges map[string]acl.Privilege
	if len(privilegeIds) > 0 {
		privileges = map[string]acl.Privilege{}
		for _, pid := range privilegeIds {
			if len(pid) > 0 {
				privileges[pid] = acl.NewStdPrivilege(pid)
			}
		}
	}

	if s.route == nil {
		s.route = map[string]map[string]*apiHandler{}
	}
	if s.route[method] == nil {
		s.route[method] = map[string]*apiHandler{}
	}
	s.route[method][endpoint] = &apiHandler{privileges: privileges, handle: handle}
}

func Register(apis *APIService) {
	apisMux.Register(apis)
}
