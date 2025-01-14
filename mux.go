package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Handle defines the API handle
type Handle func(ctx *Context) any

// Mux is a http.Handler with middlewares and routes.
type Mux struct {
	contextPool sync.Pool
	writerPool  sync.Pool
	middlewares []Handle
	router      *http.ServeMux
}

// New returns a new Mux.
func New() *Mux {
	return &Mux{
		contextPool: sync.Pool{
			New: func() any {
				return &Context{}
			},
		},
		writerPool: sync.Pool{
			New: func() any {
				return &rexWriter{}
			},
		},
	}
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
	// create the router on demand
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
	ctx := a.newContext(r)
	defer a.recycleContext(ctx)

	wr := a.newWriter(ctx, w)
	defer a.recycleWriter(wr)
	defer wr.Close()

	ctx.W = wr
	ctx.header = w.Header()
	ctx.header.Set("Connection", "keep-alive")

	if r.Method != "OPTIONS" {
		startTime := time.Now()
		defer func() {
			if ctx.accessLogger != nil {
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
	}

	defer func() {
		if v := recover(); v != nil {
			if err, ok := v.(*invalid); ok {
				ctx.W.WriteHeader(err.code)
				ctx.W.Write([]byte(err.message))
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
			ctx.W.WriteHeader(500)
			ctx.W.Write([]byte("Internal Server Error"))
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

// newContext returns a new Context from the pool.
func (a *Mux) newContext(r *http.Request) (ctx *Context) {
	ctx = a.contextPool.Get().(*Context)
	ctx.R = r
	ctx.sessionPool = defaultSessionPool
	ctx.sessionIdHandler = defaultSessionIdHandler
	ctx.logger = defaultLogger
	return
}

// recycleContext puts a Context back to the pool.
func (a *Mux) recycleContext(ctx *Context) {
	ctx.R = nil
	ctx.W = nil
	ctx.header = nil
	ctx.basicAuthUser = ""
	ctx.aclUser = nil
	ctx.session = nil
	ctx.sessionPool = nil
	ctx.sessionIdHandler = nil
	ctx.logger = nil
	ctx.accessLogger = nil
	ctx.compress = false
	a.contextPool.Put(ctx)
}

// newWriter returns a new Writer from the pool.
func (a *Mux) newWriter(ctx *Context, w http.ResponseWriter) (wr *rexWriter) {
	wr = a.writerPool.Get().(*rexWriter)
	wr.ctx = ctx
	wr.rawWriter = w
	wr.code = 200
	return
}

// recycleWriter puts a Writer back to the pool.
func (a *Mux) recycleWriter(wr *rexWriter) {
	wr.ctx = nil
	wr.code = 0
	wr.isHeaderSent = false
	wr.writeN = 0
	wr.rawWriter = nil
	wr.zWriter = nil
	a.writerPool.Put(wr)
}
