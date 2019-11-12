package router

import (
	"net/http"
	"strings"

	"github.com/ije/gox/utils"
)

// Router is a http.Handler which can be used to dispatch requests to different handler functions.
type Router struct {
	trees         map[string]*node
	validates     map[string]ValidateFn
	e404Handle    func(http.ResponseWriter, *http.Request)
	optionsHandle func(http.ResponseWriter, *http.Request)
	panicHandle   func(http.ResponseWriter, *http.Request, interface{})
}

// New returns a new initialized Router.
func New() *Router {
	return &Router{
		trees:     map[string]*node{},
		validates: map[string]ValidateFn{},
	}
}

// SetValidateFn sets param validate function.
func (router *Router) SetValidateFn(name string, fn ValidateFn) {
	if router.validates == nil {
		router.validates = map[string]ValidateFn{}
	}
	router.validates[name] = fn
}

// NotFound sets a NotFound handle.
func (router *Router) NotFound(handle http.HandlerFunc) {
	router.e404Handle = handle
}

// HandlePanic sets a panic handle.
func (router *Router) HandlePanic(handle func(http.ResponseWriter, *http.Request, interface{})) {
	router.panicHandle = handle
}

// HandleOptions sets a options handle.
func (router *Router) HandleOptions(handle func(http.ResponseWriter, *http.Request)) {
	router.optionsHandle = handle
}

// Handle registers a new request handle with the given path and method.
func (router *Router) Handle(method string, path string, handle Handle) {
	root := router.getRootNode(method)

	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	fullPath := utils.CleanPath(path)
	if fullPath == "/" {
		if root.handle == nil {
			root.handle = handle
		} else {
			panic("conflicting route: '/'")
		}
		return
	}

	pathSegs := strings.Split(strings.Trim(fullPath, "/"), "/")
	if len(pathSegs) > 0 {
		router.mapPath(root, fullPath, pathSegs, handle)
	}
}

func (router *Router) getRootNode(method string) *node {
	method = strings.ToUpper(method)
	if router.trees == nil {
		router.trees = map[string]*node{}
	}
	rootNode, ok := router.trees[method]
	if !ok {

		rootNode = &node{
			name: "/",
		}
		router.trees[method] = rootNode
	}
	return rootNode
}

func (router *Router) mapPath(n *node, fullPath string, pathSegs []string, handle Handle) {
	segs := len(pathSegs)
	if segs == 0 {
		panic("empty path segments")
	}

	fn := ""
	fs := strings.TrimSpace(pathSegs[0])
	fl := len(fs)
	isCatchAll := fl > 0 && strings.HasPrefix(fs, "*")
	isParam := fl > 0 && strings.HasPrefix(fs, ":")

	if isCatchAll {
		if segs > 1 {
			panic("bad route pattern: '" + fs + "' must be at the end of path '" + fullPath + "'")
		}
		if n.catchAllChild != nil {
			panic("conflicting route: '" + fullPath + "'")
		}
		fn = fs[1:]
		if fn == "" {
			fn = "path"
		}
		n.catchAllChild = &node{
			name:   fn,
			handle: handle,
		}
		return
	}

	validate := ""
	if !isParam {
		isParam = fl > 1 && strings.HasPrefix(fs, "{") && strings.HasSuffix(fs, "}")
		if isParam {
			s1, s2 := utils.SplitByFirstByte(fs[1:fl-1], ':')
			fn = strings.TrimSpace(s1)
			validate = strings.TrimSpace(s2)
		}
	} else {
		fn = fs[1:]
	}
	if isParam {
		if fn == "" {
			panic("bad route pattern: missing param names of path '" + fullPath + "'")
		}
		if validate != "" && len(router.validates) > 0 {
			_, ok := router.validates[validate]
			if !ok {
				panic("bad route pattern: bad validate '" + validate + "' of path '" + fullPath + "'")
			}
		}

		if n.paramChild != nil {
			n.paramChild.alias = append(n.paramChild.alias, [2]string{fn, validate})
		} else {
			n.paramChild = &node{
				name: fn,
			}
			n.paramChild.validate = validate
		}

		if segs == 1 {
			if n.paramChild.handle != nil {
				panic("conflicting route: '" + fullPath + "'")
			}
			n.paramChild.handle = handle
		} else {
			router.mapPath(n.paramChild, fullPath, pathSegs[1:], handle)
		}
		return
	}

	names := map[string]struct{}{}
	if fl > 1 && strings.HasPrefix(fs, "(") && strings.HasSuffix(fs, ")") {
		fs = fs[1 : fl-1]
		for _, name := range strings.Split(fs, "|") {
			name = strings.TrimSpace(name)
			if name != "" {
				names[name] = struct{}{}
			}
		}
		if len(names) == 0 {
			panic("bad route pattern: '" + fullPath + "'")
		}
	} else {
		names[fs] = struct{}{}
	}

	for name := range names {
		sn, ok := n.lookup(name)
		if !ok {
			sn = &node{
				name: name,
			}
			n.staticChildren = append(n.staticChildren, sn)
		}

		if segs == 1 {
			if sn.handle != nil {
				panic("conflicting route: '" + fullPath + "'")
			} else {
				sn.handle = handle
			}
		} else {
			router.mapPath(sn, fullPath, pathSegs[1:], handle)
		}
	}
}

// ServeHTTP implements the http.Handler interface.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router.panicHandle != nil {
		defer router.recover(w, r)
	}

	root, ok := router.trees[r.Method]
	if !ok {
		if r.Method == "OPTIONS" && router.optionsHandle != nil {
			router.optionsHandle(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	handle, params := router.getHandle(root, r.URL.Path)
	if handle != nil {
		handle(w, r, params)
	} else {
		if router.e404Handle != nil {
			router.e404Handle(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

func (router *Router) getHandle(root *node, path string) (Handle, Params) {
	if path == "/" {
		if root.catchAllChild != nil {
			return root.catchAllChild.handle, Params{
				{
					Key:   "path",
					Value: "/",
				},
			}
		}
		if root.handle != nil {
			return root.handle, Params{}
		}
		return nil, nil
	}

	pathLen := len(path)
	seg := ""
	segStart := 1
	parentNode := root
	params := make(Params, 0, 10)
	addParam := func(key string, value string) {
		l := len(params)
		if l == 10 {
			buf := make(Params, 10, 100)
			copy(buf, params)
			params = buf
		}
		params = params[:l+1]
		params[l].Key = key
		params[l].Value = value
	}

	for {
		if pathLen > 1 && path[pathLen-1] == '/' {
			pathLen--
		} else {
			break
		}
	}

	for i := 1; i < pathLen; i++ {
		end := i == pathLen-1
		if path[i] == '/' || end {
			if end {
				seg = path[segStart : i+1]
			} else {
				seg = path[segStart:i]
			}

			childNode, ok := parentNode.lookup(seg)

			if !ok {
				ok = parentNode.paramChild != nil
				if ok && parentNode.paramChild.validate != "" {
					var fn ValidateFn
					fn, ok = router.validates[parentNode.paramChild.validate]
					if ok {
						ok = fn(seg)
					}
					if !ok && len(parentNode.paramChild.alias) > 0 {
						for _, alia := range parentNode.paramChild.alias {
							if alia[1] == "" {
								ok = true
								break
							} else {
								fn, shi := router.validates[alia[1]]
								if shi && fn(seg) {
									ok = true
									break
								}
							}
						}
					}
				}
				if ok {
					childNode = parentNode.paramChild
					addParam(childNode.name, seg)
					if len(childNode.alias) > 0 {
						for _, alia := range childNode.alias {
							if alia[1] == "" {
								addParam(alia[0], seg)
							} else {
								fn, ok := router.validates[alia[1]]
								if ok && fn(seg) {
									addParam(alia[0], seg)
								}
							}
						}
					}
				}
			}

			if !ok {
				if parentNode.catchAllChild != nil {
					addParam(parentNode.catchAllChild.name, "/"+path[segStart:])
					return parentNode.catchAllChild.handle, params
				}
				return nil, nil
			}

			if end {
				return childNode.handle, params
			}

			parentNode = childNode
			segStart = i + 1
		}
	}

	return nil, nil
}

func (router *Router) recover(w http.ResponseWriter, r *http.Request) {
	if v := recover(); v != nil {
		router.panicHandle(w, r, v)
	}
}
