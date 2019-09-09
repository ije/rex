package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/router"
)

// Handle defines a function to handle route requests.
type Handle func(ctx *Context)

// REST is a http Handler which contains the router, middlewares and configuration settings
type REST struct {
	host string

	// Prefix to add base path at beginning of each route path
	// For example if the Prefix equals "v2", the given route path "/path" will be "/v2/path"
	prefix string

	// If enabled, errors will be sent to the client/browser,
	// this should be disable in production.
	debug bool

	// Logger to log requests
	AccessLogger Logger

	// Logger to log errors
	Logger Logger

	notFoundHandle Handle
	middlewares    []Handle
	router         *router.Router
}

// New returns a new REST
func New() *REST {
	rest := &REST{
		host: "*",
	}
	global(rest)
	return rest
}

// Host returns a nested REST with host
func (rest *REST) Host(host string) *REST {
	if host == "" {
		host = "*"
	}
	if host == rest.host {
		return rest
	}

	rest.host = host
	global(rest)
	return rest
}

// Prefix returns a nested REST with prefix
func (rest *REST) Prefix(prefix string) *REST {
	prefix = strings.TrimSpace(strings.Trim(strings.TrimSpace(prefix), "/"))
	if prefix == rest.prefix {
		return rest
	}

	rest.prefix = prefix
	global(rest)
	return rest
}

// Debug sets the debug mode
func (rest *REST) Debug(debug bool) *REST {
	rest.debug = debug
	return rest
}

// Group creates a nested REST
func (rest *REST) Group(prefix string, callback func(*REST)) {
	if callback == nil {
		return
	}

	prefix = strings.TrimSpace(strings.Trim(strings.TrimSpace(prefix), "/"))
	if prefix == "" {
		callback(rest)
		return
	}

	var s []string
	if rest.prefix != "" {
		s = append(s, rest.prefix)
	}
	s = append(s, prefix)
	middlewaresN := make([]Handle, len(rest.middlewares))
	for i, h := range rest.middlewares {
		middlewaresN[i] = h
	}
	restN := &REST{
		host:           rest.host,
		prefix:         strings.Join(s, "/"),
		AccessLogger:   rest.AccessLogger,
		Logger:         rest.Logger,
		debug:          rest.debug,
		middlewares:    middlewaresN,
		notFoundHandle: rest.notFoundHandle,
	}
	global(restN)
	callback(restN)
}

// Use appends middleware to the REST middleware stack.
func (rest *REST) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			rest.middlewares = append(rest.middlewares, handle)
		}
	}
}

// NotFound handles the requests that are not routed
func (rest *REST) NotFound(handle Handle) {
	rest.notFoundHandle = handle
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

// Handle registers a new request handle with the given method and path.
func (rest *REST) Handle(method string, path string, handles ...Handle) {
	if method == "" || path == "" || len(handles) == 0 {
		return
	}

	if rest.prefix != "" {
		path = utils.CleanPath(rest.prefix + "/" + path)
	}

	if rest.router == nil {
		rest.initRouter()
	}

	rest.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params router.Params) {
		rest.serve(w, r, params, handles...)
	})
}

func (rest *REST) serve(w http.ResponseWriter, r *http.Request, params router.Params, handles ...Handle) {
	startTime := time.Now()
	routePath := r.URL.Path
	if rest.prefix != "" {
		routePath = "/" + strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/"+rest.prefix), "/")
	}
	ctx := &Context{
		W:           &responseWriter{status: 200, rawWriter: w},
		R:           r,
		URL:         &URL{params, routePath, r.URL},
		handles:     append(rest.middlewares, handles...),
		handleIndex: -1,
		sidStore:    defaultSIDStore,
		sessionPool: defaultSessionPool,
		rest:        rest,
	}

	ctx.Next()

	if rest.AccessLogger != nil && r.Method != "OPTIONS" {
		w, ok := ctx.W.(*responseWriter)
		if ok {
			rest.AccessLogger.Printf(
				`%s %s %s %s %s %d %s "%s" %d %d %dms`,
				r.RemoteAddr,
				r.Host,
				r.Proto,
				r.Method,
				r.RequestURI,
				r.ContentLength,
				r.Referer(),
				strings.ReplaceAll(r.UserAgent(), `"`, ""),
				w.status,
				w.writed,
				time.Since(startTime)/time.Millisecond,
			)
		}
	}
}

func (rest *REST) initRouter() {
	router := router.New()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if rest.notFoundHandle != nil {
			rest.serve(w, r, nil, rest.notFoundHandle)
		} else {
			rest.serve(w, r, nil, func(ctx *Context) {
				ctx.End(404)
			})
		}
	})
	router.HandleOptions(func(w http.ResponseWriter, r *http.Request) {
		rest.serve(w, r, nil, func(ctx *Context) {
			ctx.End(http.StatusNoContent)
		})
	})
	router.HandlePanic(func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if err, ok := v.(*contextPanicError); ok {
			if rest.debug {
				http.Error(w, err.msg, 500)
			} else {
				http.Error(w, http.StatusText(500), 500)
			}
			if rest.Logger != nil {
				rest.Logger.Println("[error]", err.msg)
			}
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
		if rest.debug {
			http.Error(w, fmt.Sprintf("[panic] %v\n%s", v, buf.String()), 500)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		if rest.Logger != nil {
			rest.Logger.Printf("[panic] %v\n%s", v, buf.String())
		}
	})
	rest.router = router
}

// ServeHTTP implements the http Handler interface.
func (rest *REST) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rest.router != nil {
		rest.router.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}
