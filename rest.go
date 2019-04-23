package rex

import (
	"path"
)

type RESTHandle func(ctx *Context)

type REST struct {
	Prefix      string
	IsGlobal    bool
	middlewares []RESTHandle
	route       map[string]map[string][]RESTHandle
	inGlobal    bool
}

func New(prefixs ...string) *REST {
	return &REST{Prefix: path.Join(prefixs...), IsGlobal: true}
}

func (s *REST) Use(middlewares ...RESTHandle) {
	for _, handle := range middlewares {
		if handle != nil {
			s.middlewares = append(s.middlewares, handle)
		}
	}
}

func (s *REST) Options(endpoint string, handles ...RESTHandle) {
	s.Handle("OPTIONS", endpoint, handles)
}

func (s *REST) Head(endpoint string, handles ...RESTHandle) {
	s.Handle("HEAD", endpoint, handles)
}

func (s *REST) Get(endpoint string, handles ...RESTHandle) {
	s.Handle("GET", endpoint, handles)
}

func (s *REST) Post(endpoint string, handles ...RESTHandle) {
	s.Handle("POST", endpoint, handles)
}

func (s *REST) Put(endpoint string, handles ...RESTHandle) {
	s.Handle("PUT", endpoint, handles)
}

func (s *REST) Patch(endpoint string, handles ...RESTHandle) {
	s.Handle("PATCH", endpoint, handles)
}

func (s *REST) Delete(endpoint string, handles ...RESTHandle) {
	s.Handle("DELETE", endpoint, handles)
}

func (s *REST) Handle(method string, endpoint string, handles []RESTHandle) {
	if len(endpoint) == 0 || len(handles) == 0 {
		return
	}

	if endpoint == "*" {
		endpoint = "/*path"
	}

	if s.route == nil {
		s.route = map[string]map[string][]RESTHandle{}
	}
	if s.route[method] == nil {
		s.route[method] = map[string][]RESTHandle{}
	}

	for _, handle := range handles {
		if handle != nil {
			s.route[method][endpoint] = append(s.route[method][endpoint], handle)
		}
	}

	if s.IsGlobal && !s.inGlobal {
		gRESTs = append(gRESTs, s)
		s.inGlobal = true
	}
}

var gRESTs = []*REST{}
