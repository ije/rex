package rex

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ije/rex/session"
	"github.com/julienschmidt/httprouter"
)

// RESTHandle defines the handle of route or middleware of REST
type RESTHandle func(ctx *Context)

// REST is a http.Handler which contains the router, middlewares and configuration settings
type REST struct {
	// Prefix to add base path at beginning of each route path
	// For example if the Prefix equals "v2", the route path "/path" will be "/v2/path"
	Prefix string

	// Loggers to log accesses and errors
	AccessLogger Logger
	Logger       Logger

	// If enabled, errors will be sent to the client.
	// should be disable in production
	SendError bool

	// Configurations for httprouter
	// see https://godoc.org/github.com/julienschmidt/httprouter#Router
	RedirectTrailingSlash bool
	RedirectFixedPath     bool
	NotFound              http.Handler

	// Template to render html
	Template *template.Template

	middlewares []RESTHandle
	router      *httprouter.Router
}

var (
	gRESTs                = []*REST{}
	defaultSessionManager = session.NewMemorySessionManager(time.Hour / 2)
)

// New returns a new REST
func New(prefix ...string) *REST {
	rest := &REST{}
	if len(prefix) > 0 {
		rest.Prefix = strings.TrimSpace(prefix[0])
	}
	gRESTs = append(gRESTs, rest)
	return rest
}

// Use injects middlewares to REST
func (rest *REST) Use(middlewares ...RESTHandle) {
	for _, handle := range middlewares {
		if handle != nil {
			rest.middlewares = append(rest.middlewares, handle)
		}
	}
}

// Options is a shortcut for router.Handle("OPTIONS", path, handles)
func (rest *REST) Options(path string, handles ...RESTHandle) {
	rest.Handle("OPTIONS", path, handles...)
}

// Head is a shortcut for router.Handle("HEAD", path, handles)
func (rest *REST) Head(path string, handles ...RESTHandle) {
	rest.Handle("HEAD", path, handles...)
}

// Get is a shortcut for router.Handle("GET", path, handles)
func (rest *REST) Get(path string, handles ...RESTHandle) {
	rest.Handle("GET", path, handles...)
}

// Post is a shortcut for router.Handle("POST", path, handles)
func (rest *REST) Post(path string, handles ...RESTHandle) {
	rest.Handle("POST", path, handles...)
}

// Put is a shortcut for router.Handle("PUT", path, handles)
func (rest *REST) Put(path string, handles ...RESTHandle) {
	rest.Handle("PUT", path, handles...)
}

// Patch is a shortcut for router.Handle("PATCH", path, handles)
func (rest *REST) Patch(path string, handles ...RESTHandle) {
	rest.Handle("PATCH", path, handles...)
}

// Delete is a shortcut for router.Handle("DELETE", path, handles)
func (rest *REST) Delete(path string, handles ...RESTHandle) {
	rest.Handle("DELETE", path, handles...)
}

// Handle registers a new request handle with the given method and path.
func (rest *REST) Handle(method string, path string, handles ...RESTHandle) {
	if method == "" || path == "" || len(handles) == 0 {
		return
	}

	if path == "*" {
		path = "/*path"
	}
	if prefix := strings.Trim(rest.Prefix, "/"); prefix != "" {
		path = "/" + prefix + "/" + strings.TrimLeft(path, "/")
	}

	if rest.router == nil {
		rest.initRouter()
	}
	rest.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := &Context{
			W:              &clearResponseWriter{status: 200, rawWriter: w},
			R:              r,
			URL:            &URL{params, r.URL},
			State:          NewState(),
			startTime:      time.Now(),
			handles:        append(rest.middlewares, handles...),
			handleIndex:    -1,
			privileges:     map[string]struct{}{},
			sessionManager: defaultSessionManager,
			rest:           rest,
		}

		ctx.Next()

		if rest.AccessLogger != nil {
			w, ok := ctx.W.(*clearResponseWriter)
			if ok {
				rest.AccessLogger.Printf(
					`%s %s %s %s %s %d "%s" "%s" %d %d %dms`,
					r.RemoteAddr,
					r.Host,
					r.Proto,
					r.Method,
					r.RequestURI,
					r.ContentLength,
					strings.ReplaceAll(r.Referer(), `"`, "'"),
					strings.ReplaceAll(r.UserAgent(), `"`, "'"),
					w.status,
					w.writed,
					time.Since(ctx.startTime)/time.Millisecond,
				)
			}
		}
	})
}

func (rest *REST) initRouter() {
	router := httprouter.New()
	router.RedirectTrailingSlash = rest.RedirectTrailingSlash
	router.RedirectFixedPath = rest.RedirectFixedPath
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rest.NotFound != nil {
			rest.NotFound.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(404), 404)
		}
	})
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if err, ok := v.(*ctxPanicError); ok {
			if rest.SendError {
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
		for i := 5; ; i++ {
			pc, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			fmt.Fprint(buf, "\t", strings.TrimSpace(runtime.FuncForPC(pc).Name()), " ", file, ":", line, "\n")
		}
		if rest.SendError {
			http.Error(w, fmt.Sprintf("[panic] %v\n%s", v, buf.String()), 500)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		if rest.Logger != nil {
			rest.Logger.Printf("[panic] %v\n%s", v, buf.String())
		}
	}
	rest.router = router
}

// ServeHTTP implements the http.Handler interface.
func (rest *REST) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rest.router != nil {
		rest.router.ServeHTTP(w, r)
	}
}
