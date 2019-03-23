package rex

type APIHandle func(ctx *Context)
type MiddlewareHandle func(ctx *Context, next func())

type apiHandler struct {
	handle     APIHandle
	privileges map[string]struct{}
}

type APIService struct {
	Prefix      string
	middlewares []MiddlewareHandle
	route       map[string]map[string]*apiHandler
	registered  bool
}

func NewAPIService() *APIService {
	apis := &APIService{}
	return apis
}

func NewAPIServiceWithPrefix(prefix string) *APIService {
	apis := &APIService{Prefix: prefix}
	return apis
}

func (s *APIService) Use(middleware MiddlewareHandle) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *APIService) UseCORS(cors *CORS) {
	if cors != nil {
		s.middlewares = append(s.middlewares, func(ctx *Context, next func()) {
			cors.Apply(ctx.W)
			next()
		})
	}
}

func (s *APIService) Options(endpoint string, cors *CORS) {
	if cors != nil {
		s.register("OPTIONS", endpoint, func(ctx *Context) {
			cors.Apply(ctx.W)
			ctx.W.WriteHeader(204)
		}, nil)
	}
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

var globalAPIServices = []*APIService{}

func (s *APIService) register(method string, endpoint string, handle APIHandle, privilegeIds []string) {
	if len(endpoint) == 0 || handle == nil {
		return
	}

	if endpoint == "*" {
		endpoint = "/*path"
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

	if !s.registered {
		s.registered = true
		globalAPIServices = append(globalAPIServices, s)
	}
}
