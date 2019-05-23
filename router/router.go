package router

import (
	"net/http"
	"strings"
	"sync"

	"github.com/ije/gox/utils"
)

// Router is a http.Handler which can be used to dispatch requests to different handler functions.
type Router struct {
	lock        sync.RWMutex
	trees       map[string]*node
	validates   Validates
	notFound    func(http.ResponseWriter, *http.Request)
	panicHandle func(http.ResponseWriter, *http.Request, interface{})
}

// New returns a new initialized Router.
func New() *Router {
	return &Router{
		trees:     map[string]*node{},
		validates: Validates{},
	}
}

// Validates sets param validates.
func (router *Router) Validates(validates Validates) {
	router.lock.Lock()
	defer router.lock.Unlock()

	if validates == nil {
		validates = Validates{}
	}
	router.validates = validates
}

// Panic sets a NotFound handle.
func (router *Router) NotFound(handle http.HandlerFunc) {
	router.notFound = handle
}

// Panic sets a panic handle.
func (router *Router) Panic(handle func(http.ResponseWriter, *http.Request, interface{})) {
	router.panicHandle = handle
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
			panic("duplicate root route: '/'")
		}
		return
	}

	pathSegs := strings.Split(strings.TrimRight(fullPath, "/"), "/")[1:]
	if len(pathSegs) > 0 {
		router.mapPath(root, fullPath, pathSegs, handle)
	}
}

func (router *Router) getRootNode(method string) *node {
	method = strings.ToUpper(method)
	router.lock.RLock()
	if router.trees == nil {
		router.trees = map[string]*node{}
	}
	rootNode, ok := router.trees[method]
	router.lock.RUnlock()
	if !ok {
		rootNode = &node{
			name:           "/",
			nodeType:       root,
			staticChildren: map[string]*node{},
		}
		router.lock.Lock()
		router.trees[method] = rootNode
		router.lock.Unlock()
	}
	return rootNode
}

func (router *Router) mapPath(n *node, fullPath string, pathSegs []string, handle Handle) {
	segs := len(pathSegs)
	if segs == 0 {
		panic("empty path segments")
	}

	fs := pathSegs[0]
	fl := len(fs)
	isCatchAll := fl > 0 && strings.HasPrefix(fs, "*")
	isParam := fl > 1 && strings.HasPrefix(fs, ":")
	fn := ""
	validateName := ""
	if isCatchAll || isParam {
		fn = fs[1:]
	}
	if !isParam {
		isParam = fl > 2 && strings.HasPrefix(fs, "{") && strings.HasSuffix(fs, "}")
		if isParam {
			fn, validateName = utils.SplitByFirstByte(fs[1:fl-1], ':')
		}
	}
	if isCatchAll {
		if isCatchAll && segs > 1 {
			panic("bad route path: '" + fs + "' must be at the end of path '" + fullPath + "'")
		}
		if n.wildChild != nil {
			panic("duplicate wildcard route: '" + fullPath + "'")
		}
		if fn == "" {
			fn = "path"
		}
		n.wildChild = &node{
			name:     fn,
			nodeType: catchAll,
			handle:   handle,
		}
		return
	}

	if isParam {
		if n.wildChild != nil {
			if n.wildChild.name != fn {
				panic("duplicate wildcard route: '" + fullPath + "'")
			}
		} else {
			n.wildChild = &node{
				name:           fn,
				nodeType:       param,
				staticChildren: map[string]*node{},
			}
		}
		if validateName != "" && len(router.validates) > 0 {
			router.lock.RLock()
			validate, ok := router.validates[validateName]
			router.lock.RUnlock()
			if ok {
				n.wildChild.validate = validate
			}
		}
		if segs == 1 {
			if n.wildChild.handle != nil {
				panic("duplicate wildcard route: '" + fullPath + "'")
			}
			n.wildChild.handle = handle
		} else {
			router.mapPath(n.wildChild, fullPath, pathSegs[1:], handle)
		}
		return
	}

	n.lock.RLock()
	sn, ok := n.staticChildren[fs]
	n.lock.RUnlock()
	if !ok {
		sn = &node{
			name:           fs,
			nodeType:       static,
			staticChildren: map[string]*node{},
		}
		n.lock.Lock()
		n.staticChildren[fs] = sn
		n.lock.Unlock()
	}

	if segs == 1 {
		if sn.handle != nil {
			panic("duplicate static route: '" + fullPath + "'")
		} else {
			sn.handle = handle
		}
	} else {
		router.mapPath(sn, fullPath, pathSegs[1:], handle)
	}
}

// ServeHTTP implements the http.Handler interface.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router.panicHandle != nil {
		defer router.recover(w, r)
	}

	router.lock.RLock()
	root, ok := router.trees[r.Method]
	router.lock.RUnlock()
	if !ok {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	path := utils.CleanPath(r.URL.Path)
	if path == "/" {
		if root.wildChild != nil && root.wildChild.handle != nil {
			root.wildChild.handle(w, r, Params{
				root.wildChild.name: "/",
			})
		} else if root.handle != nil {
			root.handle(w, r, Params{})
		} else {
			router.handleNotFound(w, r)
		}
		return
	}

	// fmt.Println(path)
	pathSegs := strings.Split(path, "/")[1:]
	parentNode := root
	params := Params{}
	for len(pathSegs) > 0 {
		fs := pathSegs[0]
		// fmt.Println("+ " + fs)
		parentNode.lock.RLock()
		childNode, ok := parentNode.staticChildren[fs]
		parentNode.lock.RUnlock()
		if !ok {
			ok = parentNode.wildChild != nil
			if ok {
				childNode = parentNode.wildChild
			}
		}
		// fmt.Println("?", ok)
		if !ok {
			router.handleNotFound(w, r)
			return
		}
		if childNode.nodeType == param {
			if childNode.validate != nil {
				if !childNode.validate(fs) && len(pathSegs) == 1 {
					router.handleNotFound(w, r)
					return
				}
			}
			params[childNode.name] = fs
		} else if childNode.nodeType == catchAll {
			params[childNode.name] = "/" + strings.Join(pathSegs, "/")
		}
		if len(pathSegs) == 1 || childNode.nodeType == catchAll {
			if childNode.handle != nil {
				childNode.handle(w, r, params)
			} else {
				router.handleNotFound(w, r)
			}
			return
		}
		parentNode = childNode
		pathSegs = pathSegs[1:]
	}
}

func (router *Router) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if router.notFound != nil {
		router.notFound(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func (router *Router) recover(w http.ResponseWriter, r *http.Request) {
	if v := recover(); v != nil {
		router.panicHandle(w, r, v)
	}
}
