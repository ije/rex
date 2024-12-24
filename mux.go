package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Handle defines the API handle
type Handle func(ctx *Context) any

// Mux is a http.Handler with middlewares and routes.
type Mux struct {
	middlewares []Handle
	router      *http.ServeMux
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
	if a.router == nil {
		a.router = http.NewServeMux()
	}
	a.router.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
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
	ctx := newContext(r)
	defer recycleContext(ctx)

	wr := newWriter(ctx, w)
	defer recycleWriter(wr)

	ctx.W = wr
	ctx.Header = w.Header()
	ctx.Header.Set("Connection", "keep-alive")

	startTime := time.Now()
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
				wr.code,
				wr.writeN,
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
		if v != next {
			ctx.respondWith(v)
			return
		}
	}

	if a.router != nil {
		a.router.ServeHTTP(wr, r)
		return
	}

	if r.Method == "GET" {
		ctx.respondWith(&status{404, "Not Found"})
		return
	}

	ctx.respondWith(&status{405, "Method Not Allowed"})
}
