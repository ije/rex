package rex

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
	"github.com/julienschmidt/httprouter"
)

type Mux struct {
	*Config
	Logger         *log.Logger
	SessionManager session.Manager
	router         *httprouter.Router
}

func (mux *Mux) initRouter() *httprouter.Router {
	router := httprouter.New()
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if mux.Debug {
			http.Error(w, fmt.Sprintf("%v", v), 500)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}

		if err, ok := v.(*initSessionError); ok {
			if mux.Logger != nil {
				mux.Logger.Errorf("Init session: %s", err.msg)
			}
			return
		}

		var (
			i    = 2
			j    int
			pc   uintptr
			file string
			line int
			ok   bool
			buf  = bytes.NewBuffer(nil)
		)
		for {
			pc, file, line, ok = runtime.Caller(i)
			if ok {
				buf.WriteByte('\n')
				for j = 0; j < 34; j++ {
					buf.WriteByte(' ')
				}
				fmt.Fprint(buf, "> ", runtime.FuncForPC(pc).Name(), " ", file, ":", line)
			} else {
				break
			}
			i++
		}
		if mux.Logger != nil {
			mux.Logger.Error("[panic]", v, buf.String())
		}
	}
	if mux.Root != "" {
		router.NotFound = &staticMux{mux.Root, mux.NotFoundHandler}
	} else if mux.NotFoundHandler != nil {
		router.NotFound = mux.NotFoundHandler
	}
	return router
}

func (mux *Mux) RegisterAPIService(apis *APIService) {
	if apis == nil {
		return
	}

	if mux.router == nil {
		mux.router = mux.initRouter()
	}

	for method, route := range apis.route {
		for endpoint, handler := range route {
			var routerHandle func(string, httprouter.Handle)
			switch method {
			case "OPTIONS":
				routerHandle = mux.router.OPTIONS
			case "HEAD":
				routerHandle = mux.router.HEAD
			case "GET":
				routerHandle = mux.router.GET
			case "POST":
				routerHandle = mux.router.POST
			case "PUT":
				routerHandle = mux.router.PUT
			case "PATCH":
				routerHandle = mux.router.PATCH
			case "DELETE":
				routerHandle = mux.router.DELETE
			}
			if routerHandle == nil {
				continue
			}
			if len(apis.Prefix) > 0 {
				endpoint = path.Join("/"+strings.Trim(apis.Prefix, "/"), endpoint)
			}
			func(mux *Mux, routerHandle func(string, httprouter.Handle), endpoint string, handler *apiHandler, apis *APIService) {
				routerHandle(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
					url := &URL{params, r.URL}
					state := NewState()
					ctx := &Context{
						W:     w,
						R:     r,
						URL:   url,
						State: state,
						mux:   mux,
					}

					if len(apis.middlewares) > 0 {
						for _, use := range apis.middlewares {
							shouldContinue := false
							use(ctx, func() {
								shouldContinue = true
							})
							if !shouldContinue {
								return
							}

							// prevent user chanage the 'read-only' fields in context
							ctx.W = w
							ctx.R = r
							ctx.URL = url
							ctx.State = state
						}
					}

					if len(handler.privileges) > 0 {
						var isGranted bool
						if ctx.user != nil {
							for _, pid := range ctx.user.Privileges() {
								_, isGranted = handler.privileges[pid]
								if isGranted {
									break
								}
							}
						}
						if !isGranted {
							ctx.End(http.StatusUnauthorized)
						}
					}

					handler.handle(ctx)
				})
			}(mux, routerHandle, endpoint, handler, apis)
		}
	}
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// wrap the ResponseWriter
	w = &ResponseWriter{status: 200, rawWriter: w}

	if mux.AccessLogger != nil {
		d := time.Since(time.Now())
		defer func() {
			rw, ok := w.(*ResponseWriter)
			if ok {
				mux.AccessLogger.Printf(
					`%s %s %s %s %s %d "%s" "%s" %d %d %dms`,
					r.RemoteAddr,
					r.Host,
					r.Proto,
					r.Method,
					r.RequestURI,
					r.ContentLength,
					strings.ReplaceAll(r.Referer(), `"`, "'"),
					strings.ReplaceAll(r.UserAgent(), `"`, "'"),
					rw.status,
					rw.writedBytes,
					d/time.Millisecond,
				)
			}
		}()
	}

	wh := w.Header()
	if len(mux.CustomHTTPHeaders) > 0 {
		for key, val := range mux.CustomHTTPHeaders {
			wh.Set(key, val)
		}
	}

	wh.Set("Connection", "keep-alive")
	if len(mux.ServerName) > 0 {
		wh.Set("Server", mux.ServerName)
	} else {
		wh.Set("Server", "rex-serv")
	}

	if len(mux.HostRedirectRule) > 0 {
		code := 301 // Permanent redirect, request with GET method
		if r.Method != "GET" {
			// Temporary redirect, request with same method
			// As of Go 1.3, Go does not support status code 308.
			code = 307
		}
		if mux.HostRedirectRule == "force-www" {
			if !strings.HasPrefix(r.Host, "www.") {
				http.Redirect(w, r, path.Join("www."+r.Host, r.URL.String()), code)
				return
			}
		} else if mux.HostRedirectRule == "remove-www" {
			if strings.HasPrefix(r.Host, "www.") {
				http.Redirect(w, r, path.Join(strings.TrimPrefix(r.Host, "www."), r.URL.String()), code)
				return
			}
		}
	}

	if mux.router == nil {
		mux.router = mux.initRouter()
	}
	mux.router.ServeHTTP(w, r)
}

type staticMux struct {
	root  string
	final http.Handler
}

func (mux *staticMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rootIndexHTML := utils.CleanPath(path.Join(mux.root, "index.html"))
	file := utils.CleanPath(path.Join(mux.root, r.URL.Path))
Re:
	fi, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// 404s will fallback to /index.html
		if file != rootIndexHTML {
			file = rootIndexHTML
			goto Re
		}

		if mux.final != nil {
			mux.final.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(404), 404)
		}
	}

	if fi.IsDir() {
		file = path.Join(file, "index.html")
		goto Re
	}

	// compress text files when the size is greater than 1024 bytes
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		switch strings.ToLower(utils.FileExt(file)) {
		case "js", "css", "html", "htm", "xml", "svg", "json", "txt":
			if fi.Size() > 1024 {
				if w, ok := w.(*ResponseWriter); ok {
					gzw := newGzResponseWriter(w.rawWriter)
					defer gzw.Close()
					w.rawWriter = gzw
				}
			}
		}
	}

	http.ServeFile(w, r, file)
}

type ResponseWriter struct {
	status      int
	writedBytes int
	rawWriter   http.ResponseWriter
}

func (w *ResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

func (w *ResponseWriter) WriteHeader(status int) {
	w.status = status
	if w.writedBytes == 0 {
		w.rawWriter.WriteHeader(status)
	}
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	n, err = w.rawWriter.Write(p)
	w.writedBytes += n
	return
}

func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.rawWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("Response does not implement the http.Hijacker")
}

type gzResponseWriter struct {
	gzipWriter io.WriteCloser
	rawWriter  http.ResponseWriter
}

func newGzResponseWriter(w http.ResponseWriter) (gzw *gzResponseWriter) {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	gzipWriter, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	gzw = &gzResponseWriter{gzipWriter, w}
	return
}

func (w *gzResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

func (w *gzResponseWriter) WriteHeader(status int) {
	w.rawWriter.WriteHeader(status)
}

func (w *gzResponseWriter) Write(p []byte) (int, error) {
	return w.gzipWriter.Write(p)
}

func (w *gzResponseWriter) Close() error {
	return w.gzipWriter.Close()
}
