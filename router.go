package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/gox/utils"
)

// Handle defines the API handle
type Handle func(ctx *Context) interface{}

// Router is a http.Handler with middlewares and routes.
type Router struct {
	middlewares []Handle
	routes      map[string]*node
	paramsPool  sync.Pool
	maxParams   uint16
}

// Use appends middlewares to current APIS middleware stack.
func (a *Router) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			a.middlewares = append(a.middlewares, handle)
		}
	}
}

// AddRoute adds a route.
func (a *Router) AddRoute(method string, pattern string, handle Handle) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(pattern) == 0 || pattern[0] != '/' {
		panic("path must begin with '/' in path '" + pattern + "'")
	}
	if handle == nil {
		panic("handle must not be nil")
	}

	if a.routes == nil {
		a.routes = map[string]*node{}
	}
	root := a.routes[method]
	if root == nil {
		root = new(node)
		a.routes[method] = root
	}
	root.addRoute(pattern, handle)

	// Update maxParams
	if paramsCount := countParams(pattern); paramsCount > a.maxParams {
		a.maxParams = paramsCount
	}

	// Lazy-init paramsPool alloc func
	if a.paramsPool.New == nil && a.maxParams > 0 {
		a.paramsPool.New = func() interface{} {
			ps := make(Params, 0, a.maxParams)
			return &ps
		}
	}
}

// ServeHTTP implements the http Handler.
func (a *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	wr := &responseWriter{status: 200, httpWriter: w}
	path := &Path{
		raw: utils.CleanPath(r.URL.Path),
	}
	form := &Form{r}
	store := &Store{}
	ctx := &Context{
		W:                wr,
		R:                r,
		Path:             path,
		Form:             form,
		Store:            store,
		sessionIdHandler: defaultSessionIdHandler,
		sessionPool:      defaultSessionPool,
		logger:           &log.Logger{},
	}

	defer func() {
		wr.Close()

		if ctx.accessLogger != nil && r.Method != "OPTIONS" {
			ref := r.Referer()
			if ref == "" {
				ref = "-"
			}
			ctx.accessLogger.Printf(
				`%s %s %s %s %s %d %s "%s" %d %d %dms`,
				ctx.RemoteIP(),
				r.Host,
				r.Proto,
				r.Method,
				r.RequestURI,
				r.ContentLength,
				ref,
				strings.ReplaceAll(r.UserAgent(), `"`, `\"`),
				wr.status,
				wr.written,
				time.Since(startTime)/time.Millisecond,
			)
		}
	}()

	defer func() {
		if v := recover(); v != nil {
			if err, ok := v.(*recoverError); ok {
				ctx.error(&Error{err.status, err.message})
				return
			}

			buf := bytes.NewBuffer(nil)
			for i := 3; ; i++ {
				pc, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				fmt.Fprint(buf, "\t", strings.TrimSpace(runtime.FuncForPC(pc).Name()), " ", file, ":", line, "\n")
			}

			if ctx.logger != nil {
				ctx.logger.Printf("[panic] %v\n%s", v, buf.String())
			}
			ctx.error(&Error{500, http.StatusText(500)})
		}
	}()

	for _, handle := range a.middlewares {
		ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = wr, r, path, form, store
		v := handle(ctx)
		if v != nil {
			ctx.end(v)
			return
		}
	}

	if root := a.routes[r.Method]; root != nil {
		handle, ps, _ := root.getValue(path.raw, a.getParams)
		if handle != nil {
			if ps != nil {
				path.Params = *ps
			}
			v := handle(ctx)
			if ps != nil {
				a.putParams(ps)
			}
			if v != nil {
				ctx.end(v)
				return
			}
		}
	}

	if r.Method == "GET" {
		ctx.end(&statusPlayload{404, "Not Found"})
		return
	}

	ctx.end(&statusPlayload{405, "Method Not Allowed"})
}

func (a *Router) getParams() *Params {
	ps, _ := a.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0] // reset slice
	return ps
}

func (a *Router) putParams(ps *Params) {
	if ps != nil {
		a.paramsPool.Put(ps)
	}
}
