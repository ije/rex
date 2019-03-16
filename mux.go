package rex

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	App               *App
	Debug             bool
	ServerName        string
	CustomHTTPHeaders map[string]string
	SessionCookieName string
	HostRedirectRule  string
	SessionManager    session.Manager
	AccessLogger      *log.Logger
	Logger            *log.Logger
	router            *httprouter.Router
}

func (mux *Mux) initRouter() *httprouter.Router {
	router := httprouter.New()
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if mux.Debug {
			http.Error(w, fmt.Sprintf("%v", v), http.StatusInternalServerError)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	if mux.App != nil {
		router.NotFound = &AppMux{mux.App}
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
			if mux.App != nil {
				endpoint = path.Join("/api", endpoint)
			}
			func(mux *Mux, routerHandle func(string, httprouter.Handle), endpoint string, handler *apiHandler, apis *APIService) {
				routerHandle(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
					url := &URL{params, r.URL}
					ctx := &Context{
						App:            mux.App,
						ResponseWriter: w,
						Request:        r,
						URL:            url,
						mux:            mux,
					}

					if len(apis.middlewares) > 0 {
						for _, use := range apis.middlewares {
							shouldEnd := false
							use(ctx, func() {
								shouldEnd = true
							})
							if shouldEnd {
								return
							}

							// prevent user chanage the 'read-only' fields in context
							ctx.App = mux.App
							ctx.ResponseWriter = w
							ctx.Request = r
							ctx.URL = url
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
	if mux.AccessLogger != nil {
		d := time.Since(time.Now())
		defer func() {
			rw, ok := w.(*ResponseWriter)
			if ok {
				status, writedBytes := rw.Status()
				mux.AccessLogger.Printf(`%s %s %s %s %s %d "%s" "%s" %d %d %dms`, r.RemoteAddr, r.Host, r.Proto, r.Method, r.RequestURI, r.ContentLength, strings.Replace(r.Referer(), `"`, "'", -1), strings.Replace(r.UserAgent(), `"`, "'", -1), status, writedBytes, d/time.Millisecond)
			}
		}()
	}

	// wrap http ResponseWriter
	w = &ResponseWriter{status: 200, rawWriter: w}

	wh := w.Header()
	if len(mux.CustomHTTPHeaders) > 0 {
		for key, val := range mux.CustomHTTPHeaders {
			wh.Set(key, val)
		}
	}
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", mux.ServerName)

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

type AppMux struct {
	*App
}

func (mux *AppMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.App == nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	if mux.debug {
		if mux.debugProcess != nil {
			remote, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", mux.debugPort))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(remote)
			proxy.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}

	// todo: app ssr

	// Serve app dist files
	filePath := utils.CleanPath(path.Join(mux.Dir(), r.URL.Path))
Lookup:
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 404s will fallback to /index.html
			if topIndexHTML := path.Join(mux.Dir(), "index.html"); filePath != topIndexHTML {
				filePath = topIndexHTML
				goto Lookup
			}
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if fi.IsDir() {
		filePath = path.Join(filePath, "index.html")
		goto Lookup
	}

	// compress text files when the size is greater than 1024 bytes
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		switch strings.ToLower(utils.FileExt(filePath)) {
		case "js", "css", "html", "htm", "xml", "svg", "json", "txt":
			if fi.Size() > 1024 {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Vary", "Accept-Encoding")
				gzw, err := newGzResponseWriter(w, gzip.BestSpeed)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				defer gzw.Close()
				w = &ResponseWriter{status: 200, rawWriter: gzw}
			}
		}
	}

	http.ServeFile(w, r, filePath)
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
	w.rawWriter.WriteHeader(status)
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	n, err = w.rawWriter.Write(p)
	w.writedBytes += n
	return
}

func (w *ResponseWriter) Status() (status int, writedBytes int) {
	return w.status, w.writedBytes
}

type gzResponseWriter struct {
	gzipWriter io.WriteCloser
	rawWriter  http.ResponseWriter
}

func newGzResponseWriter(w http.ResponseWriter, speed int) (gzw *gzResponseWriter, err error) {
	gzipWriter, err := gzip.NewWriterLevel(w, speed)
	if err != nil {
		return
	}

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
