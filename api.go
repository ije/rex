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

// APIHandler is a query/mutation style API http Handler
type APIHandler struct {
	// Prefix to add prefix for each api path, like "v2"
	Prefix string

	middlewares []Handle
	queries     map[string][]Handle
	mutations   map[string][]Handle
}

// Use appends middlewares to current APIS middleware stack.
func (a *APIHandler) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			a.middlewares = append(a.middlewares, handle)
		}
	}
}

// Query adds a query api
func (a *APIHandler) Query(endpoint string, handles ...Handle) {
	if a.queries == nil {
		a.queries = map[string][]Handle{}
	}
	for _, handle := range handles {
		if handle != nil {
			a.queries[endpoint] = append(a.queries[endpoint], handle)
		}
	}
}

// Mutation adds a mutation api
func (a *APIHandler) Mutation(endpoint string, handles ...Handle) {
	if a.mutations == nil {
		a.mutations = map[string][]Handle{}
	}
	for _, handle := range handles {
		if handle != nil {
			a.mutations[endpoint] = append(a.mutations[endpoint], handle)
		}
	}
}

// ServeHTTP implements the http Handler.
func (a *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	wr := &responseWriter{status: 200, rawWriter: w}
	form := &Form{r}
	store := &Store{}
	ctx := &Context{
		W:           wr,
		R:           r,
		Form:        form,
		Store:       store,
		sidStore:    defaultSIDStore,
		sessionPool: defaultSessionPool,
		logger:      &log.Logger{},
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
				ctx.json(&Error{err.status, err.message}, err.status)
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
			ctx.json(&Error{500, http.StatusText(500)}, 500)
		}
	}()

	var apiHandles map[string][]Handle
	switch r.Method {
	case "GET":
		apiHandles = a.queries
	case "POST":
		apiHandles = a.mutations
	default:
		ctx.json(
			&Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)},
			http.StatusMethodNotAllowed,
		)
		return
	}

	pathname := r.URL.Path
	if a.Prefix != "" {
		pathname = strings.TrimPrefix(pathname, "/"+strings.Trim(a.Prefix, "/"))
	}
	path := &Path{
		segments: strings.Split(utils.CleanPath(pathname), "/")[1:],
	}

	for _, handle := range a.middlewares {
		ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = wr, r, path, form, store
		v := handle(ctx)
		if v != nil {
			ctx.end(v)
			return
		}
	}

	var handles []Handle
	var ok bool
	if len(path.segments) > 0 {
		handles, ok = apiHandles[path.segments[0]]
	}
	if !ok {
		handles, ok = apiHandles["*"]
	}
	if !ok {
		ctx.json(&Error{404, "not found"}, 404)
		return
	}

	for _, handle := range handles {
		if len(ctx.acl) > 0 {
			var isGranted bool
			if ctx.aclUser != nil {
				for _, id := range ctx.aclUser.Permissions() {
					_, isGranted = ctx.acl[id]
					if isGranted {
						break
					}
				}
			}
			if !isGranted {
				ctx.json(
					&Error{http.StatusForbidden, http.StatusText(http.StatusForbidden)},
					http.StatusForbidden,
				)
				return
			}
		}

		ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = wr, r, path, form, store
		v := handle(ctx)
		if v != nil {
			ctx.end(v)
			return
		}
	}
}
