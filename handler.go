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
	prefix      string
	middlewares []Handle
	queries     map[string][]Handle
	mutations   map[string][]Handle
}

// Prefix adds prefix for each api path, like "v2"
func (a *Handler) Prefix(prefix string) *Handler {
	a.prefix = utils.CleanPath(prefix)
	return a
}

// Use appends middlewares to current APIS middleware stack.
func (a *Handler) Use(middlewares ...Handle) {
	for _, handle := range middlewares {
		if handle != nil {
			a.middlewares = append(a.middlewares, handle)
		}
	}
}

// Query adds a query api
func (a *Handler) Query(endpoint string, handles ...Handle) {
	endpoint = utils.CleanPath(endpoint)
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
func (a *Handler) Mutation(endpoint string, handles ...Handle) {
	endpoint = utils.CleanPath(endpoint)
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
func (a *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var apiHandles map[string][]Handle
	switch r.Method {
	case "GET":
		apiHandles = a.queries
	case "POST":
		apiHandles = a.mutations
	default:
		ctx.error(&Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
		return
	}

	pathname := utils.CleanPath(r.URL.Path)
	if a.prefix != "/" {
		pathname = strings.TrimPrefix(pathname, a.prefix)
	}
	path := &Path{
		raw:      pathname,
		segments: strings.Split(pathname[1:], "/"),
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
		handles, ok = apiHandles[pathname]
	}
	if !ok {
		for p, a := range apiHandles {
			if strings.HasSuffix(p, "/*") && strings.HasPrefix(pathname, p[:len(p)-1]) {
				handles = a
				ok = true
				break
			}
		}
	}

	if !ok {
		ctx.error(&Error{404, "endpoint not found"})
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
				ctx.error(&Error{http.StatusForbidden, http.StatusText(http.StatusForbidden)})
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
