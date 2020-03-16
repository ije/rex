package rex

import (
	"net/http"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/router"
)

// Handle defines a REST handle
type Handle func(ctx *Context)

// REST is REST-based router
type REST struct {
	// prefix to add base path at beginning of each route path
	// for example if the Prefix equals "v2", the given route path "/path" will route "/v2/path"
	Prefix string

	middlewares []Handle
}

// New returns a new REST
func New(args ...string) *REST {
	var prefix string
	if len(args) > 0 {
		prefix = strings.TrimSpace(strings.Trim(strings.TrimSpace(args[0]), "/"))
	}
	rest := &REST{
		Prefix: prefix,
	}
	return rest
}

// Group creates a nested REST
func (rest *REST) Group(prefix string, callback func(*REST)) *REST {
	prefix = strings.TrimSpace(strings.Trim(strings.TrimSpace(prefix), "/"))
	if prefix == "" {
		if callback != nil {
			callback(rest)
		}
		return rest
	}

	if rest.Prefix != "" {
		prefix = rest.Prefix + "/" + prefix
	}
	middlewaresCopy := make([]Handle, len(rest.middlewares))
	for i, h := range rest.middlewares {
		middlewaresCopy[i] = h
	}
	childRest := &REST{
		Prefix:      prefix,
		middlewares: middlewaresCopy,
	}
	if callback != nil {
		callback(childRest)
	}
	return childRest
}

// Use appends middlewares to current REST middleware stack.
func (rest *REST) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			rest.middlewares = append(rest.middlewares, handle)
		}
	}
}

// Options is a shortcut for rest.Handle("OPTIONS", path, handles)
func (rest *REST) Options(path string, handles ...Handle) {
	rest.Handle("OPTIONS", path, handles...)
}

// Head is a shortcut for rest.Handle("HEAD", path, handles)
func (rest *REST) Head(path string, handles ...Handle) {
	rest.Handle("HEAD", path, handles...)
}

// Get is a shortcut for rest.Handle("GET", path, handles)
func (rest *REST) Get(path string, handles ...Handle) {
	rest.Handle("GET", path, handles...)
}

// Post is a shortcut for rest.Handle("POST", path, handles)
func (rest *REST) Post(path string, handles ...Handle) {
	rest.Handle("POST", path, handles...)
}

// Put is a shortcut for rest.Handle("PUT", path, handles)
func (rest *REST) Put(path string, handles ...Handle) {
	rest.Handle("PUT", path, handles...)
}

// Patch is a shortcut for rest.Handle("PATCH", path, handles)
func (rest *REST) Patch(path string, handles ...Handle) {
	rest.Handle("PATCH", path, handles...)
}

// Delete is a shortcut for rest.Handle("DELETE", path, handles)
func (rest *REST) Delete(path string, handles ...Handle) {
	rest.Handle("DELETE", path, handles...)
}

// Trace is a shortcut for rest.Handle("TRACE", path, handles)
func (rest *REST) Trace(path string, handles ...Handle) {
	rest.Handle("TRACE", path, handles...)
}

// Handle handles requests that match the method and path
func (rest *REST) Handle(method string, path string, handles ...Handle) {
	if method == "" || path == "" || len(handles) == 0 {
		return
	}

	if rest.Prefix != "" {
		path = utils.CleanPath(rest.Prefix + "/" + path)
	}

	defaultRouter.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params router.Params) {
		rest.serve(w, r, params, handles...)
	})
}

func (rest *REST) serve(w http.ResponseWriter, r *http.Request, params router.Params, handles ...Handle) {
	startTime := time.Now()
	routePath := r.URL.Path
	if rest.Prefix != "" {
		routePath = "/" + strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/"+rest.Prefix), "/")
	}
	wr := &responseWriter{status: 200, rawWriter: w}
	ctx := &Context{
		W:           wr,
		R:           r,
		URL:         &URL{params, routePath, r.URL},
		Form:        &Form{r},
		handles:     append(rest.middlewares, handles...),
		handleIndex: -1,
		sidManager:  defaultSIDManager,
		sessionPool: defaultSessionPool,
	}

	ctx.Next()

	if gzw, ok := wr.rawWriter.(*gzipResponseWriter); ok {
		gzw.Close()
	}

	if accessLogger != nil && r.Method != "OPTIONS" {
		accessLogger.Printf(
			`%s %s %s %s %s %d %s "%s" %d %d %dms`,
			r.RemoteAddr,
			r.Host,
			r.Proto,
			r.Method,
			r.RequestURI,
			r.ContentLength,
			r.Referer(),
			strings.ReplaceAll(r.UserAgent(), `"`, ""),
			wr.status,
			wr.writed,
			time.Since(startTime)/time.Millisecond,
		)
	}
}
