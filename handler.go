package rex

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/gox/utils"
)

// Handle defines the API handle
type Handle func(ctx *Context) interface{}

// Handler is a query/mutation style API http Handler
type Handler struct {
	middlewares []Handle
}

// Use appends middlewares to current APIS middleware stack.
func (a *Handler) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			a.middlewares = append(a.middlewares, handle)
		}
	}
}

// ServeHTTP implements the http Handler.
func (a *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	wr := &responseWriter{status: 200, httpWriter: w}
	form := &Form{r}
	store := &Store{}
	ctx := &Context{
		W:                wr,
		R:                r,
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

	pathname := utils.CleanPath(r.URL.Path)
	path := &Path{
		raw:      pathname,
		segments: strings.Split(pathname[1:], "/"),
	}

	for _, handle := range a.middlewares {
		v := handle(ctx)
		if v != nil {
			ctx.end(v)
			return
		}
		ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = wr, r, path, form, store
	}

	if r.Method == "GET" {
		ctx.end(&statusPlayload{404, "Not Found"})
		return
	}

	ctx.end(&statusPlayload{405, "Method Not Allowed"})
}
