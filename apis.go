package wsx

import (
	"strconv"
	"strings"
)

type APIService struct {
	Prefix      string
	middlewares []APIHandle
	route       map[string]map[string]*apiHandler
}

func NewAPIService() *APIService {
	apis := &APIService{}
	return apis
}

func NewAPIServiceWithPrefix(prefix string) *APIService {
	apis := &APIService{Prefix: prefix}
	return apis
}

type apiHandler struct {
	handle     APIHandle
	privileges map[string]struct{} // set
}

type APIHandle func(*Context)

func (s *APIService) Use(middleware APIHandle) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *APIService) Options(endpoint string, cors *CORS) {
	if cors == nil {
		return
	}

	s.register("OPTIONS", endpoint, func(ctx *Context) {
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

func (s *APIService) Head(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("HEAD", endpoint, handle, privilegeIds)
}

func (s *APIService) Get(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("GET", endpoint, handle, privilegeIds)
}

func (s *APIService) Post(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("POST", endpoint, handle, privilegeIds)
}

func (s *APIService) Put(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("PUT", endpoint, handle, privilegeIds)
}

func (s *APIService) Patch(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("PATCH", endpoint, handle, privilegeIds)
}

func (s *APIService) Delete(endpoint string, handle APIHandle, privilegeIds ...string) {
	s.register("DELETE", endpoint, handle, privilegeIds)
}

func (s *APIService) register(method string, endpoint string, handle APIHandle, privilegeIds []string) {
	if len(endpoint) == 0 || handle == nil {
		return
	}

	var privileges map[string]struct{}
	if len(privilegeIds) > 0 {
		privileges = map[string]struct{}{}
		for _, pid := range privilegeIds {
			if len(pid) > 0 {
				privileges[pid] = struct{}{}
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
