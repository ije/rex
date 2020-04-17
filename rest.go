package rex

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/router"
)

// Handle defines a REST handle
type Handle func(ctx *Context)

// REST is REST-based router
type REST struct {
	// BasePath to add base path at beginning of each route path
	// for example if the BasePath equals "/v2", the given route path "/path" will route "/v2/path"
	BasePath string

	middlewares []Handle
	router      *router.Router
}

// New returns a new REST
func New(base string) *REST {
	rest := &REST{
		BasePath: utils.CleanPath(base),
		router:   router.New(),
	}
	rest.router.HandleOptions(func(w http.ResponseWriter, r *http.Request) {
		rest.serve(w, r, nil, func(ctx *Context) {
			ctx.End(http.StatusNoContent)
		})
	})
	rest.router.HandlePanic(func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if err, ok := v.(*contextPanicError); ok {
			rest.serve(w, r, nil, func(ctx *Context) {
				ctx.Error(err.message, err.code)
			})
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

		rest.serve(w, r, nil, func(ctx *Context) {
			ctx.Error(fmt.Sprintf("[panic] %v\n%s", v, buf.String()), 500)
		})
	})
	return rest
}

// Group creates a nested REST
func (rest *REST) Group(path string, callback func(*REST)) *REST {
	BasePath := utils.CleanPath(rest.BasePath + "/" + path)
	if BasePath == rest.BasePath {
		return rest
	}

	middlewaresCopy := make([]Handle, len(rest.middlewares))
	for i, h := range rest.middlewares {
		middlewaresCopy[i] = h
	}
	childRest := &REST{
		BasePath:    BasePath,
		middlewares: middlewaresCopy,
		router:      rest.router,
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

// NotFound sets a NotFound handle.
func (rest *REST) NotFound(handle Handle) {
	rest.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		rest.serve(w, r, nil, handle)
	})
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

	path = utils.CleanPath(rest.BasePath + "/" + path)
	rest.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params router.Params) {
		rest.serve(w, r, params, handles...)
	})
}

func (rest *REST) serve(w http.ResponseWriter, r *http.Request, params router.Params, handles ...Handle) {
	startTime := time.Now()
	routePath := "/" + strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, rest.BasePath), "/")
	wr := &responseWriter{status: 200, rawWriter: w}
	ctx := &Context{
		W:           wr,
		R:           r,
		URL:         &URL{params, routePath, r.URL},
		Form:        &Form{r},
		handles:     append(rest.middlewares, handles...),
		handleIndex: -1,
		sidStore:    defaultSIDStore,
		sessionPool: defaultSessionPool,
		sendError:   false,
		errorType:   "text",
		logger:      log.New(os.Stderr, "", log.LstdFlags),
	}

	ctx.Next()

	if gzw, ok := wr.rawWriter.(*gzipResponseWriter); ok {
		gzw.Close()
	}

	if ctx.accessLogger != nil && r.Method != "OPTIONS" {
		ctx.accessLogger.Printf(
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

func (rest *REST) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest.router.ServeHTTP(w, r)
}
