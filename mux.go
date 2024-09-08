package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ije/gox/log"
)

// Handle defines the API handle
type Handle func(ctx *Context) interface{}

// Mux is a http.Handler with middlewares and routes.
type Mux struct {
	middlewares []Handle
	mux         *http.ServeMux
}

// Use appends middlewares to current APIS middleware stack.
func (a *Mux) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			a.middlewares = append(a.middlewares, handle)
		}
	}
}

// AddRoute adds a route.
func (a *Mux) AddRoute(pattern string, handle Handle) {
	if a.mux == nil {
		a.mux = http.NewServeMux()
	}
	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		wr, ok := w.(*rexWriter)
		if ok {
			v := handle(wr.ctx)
			if v != nil {
				wr.ctx.respondWith(v)
			}
		}
	})
}

// ServeHTTP implements the http Handler.
func (a *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	store := &Store{}
	ctx := &Context{
		R:                r,
		Store:            store,
		sessionIdHandler: defaultSessionIdHandler,
		sessionPool:      defaultSessionPool,
		logger:           &log.Logger{},
	}
	wr := &rexWriter{ctx: ctx, status: 200, httpWriter: w}
	ctx.W = wr

	header := w.Header()
	header.Set("Connection", "keep-alive")
	header.Set("Server", "rex")

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
			if err, ok := v.(*invalid); ok {
				ctx.respondWithError(&Error{err.status, err.message})
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
			ctx.respondWithError(&Error{500, http.StatusText(500)})
		}
	}()

	for _, handle := range a.middlewares {
		v := handle(ctx)
		if v != nil {
			ctx.respondWith(v)
			return
		}
	}

	if a.mux != nil {
		a.mux.ServeHTTP(wr, r)
		return
	}

	if r.Method == "GET" {
		ctx.respondWith(&statusd{404, "Not Found"})
		return
	}

	ctx.respondWith(&statusd{405, "Method Not Allowed"})
}

// PathValue returns the value for the named path wildcard in the [ServeMux] pattern
// that matched the request.
// It returns the empty string if the request was not matched against a pattern
// or there is no such wildcard in the pattern.
func (ctx *Context) PathValue(key string) string {
	return ctx.R.PathValue(key)
}

// SetPathValue sets name to value, so that subsequent calls to r.PathValue(name)
// return value.
func (ctx *Context) SetPathValue(key string, value string) {
	ctx.R.SetPathValue(key, value)
}
