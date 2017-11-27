package webx

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

	"github.com/ije/gox/utils"
	"github.com/julienschmidt/httprouter"
)

type ApisMux struct {
	router *httprouter.Router
	apiss  []*APIService
}

func (mux *ApisMux) RegisterApis(apis *APIService) {
	if apis == nil {
		return
	}

	mux.apiss = append(mux.apiss, apis)
}

func (mux *ApisMux) InitRouter(app *App) {
	if mux.router != nil {
		return
	}

	router := httprouter.New()
	mux.router = router
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if config.Debug {
			http.Error(w, fmt.Sprintf("%v", v), http.StatusInternalServerError)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		if err, ok := v.(*initSessionError); ok {
			log.Errorf("Init session: %s", err.msg)
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
		log.Error("[panic]", v, buf.String())
	}
	if app != nil {
		router.NotFound = &AppMux{app}
	}

	for _, apis := range mux.apiss {
		for method, route := range apis.route {
			for endpoint, handler := range route {
				var routerHandle func(string, httprouter.Handle)
				switch method {
				case "OPTIONS":
					routerHandle = router.OPTIONS
				case "HEAD":
					routerHandle = router.HEAD
				case "GET":
					routerHandle = router.GET
				case "POST":
					routerHandle = router.POST
				case "PUT":
					routerHandle = router.PUT
				case "PATCH":
					routerHandle = router.PATCH
				case "DELETE":
					routerHandle = router.DELETE
				}
				if routerHandle == nil {
					continue
				}
				if len(apis.Prefix) > 0 {
					endpoint = path.Join("/"+strings.Trim(apis.Prefix, "/"), endpoint)
				}
				if app != nil {
					endpoint = path.Join("/api", endpoint)
				}
				routerHandle(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
					ctx := &Context{
						App:            app,
						ResponseWriter: w,
						Request:        r,
						URL:            &URL{params, r.URL},
					}

					if len(apis.middlewares) > 0 {
						for _, use := range apis.middlewares {
							use(ctx)
						}
					}

					ctx.App = app
					ctx.ResponseWriter = w
					ctx.Request = r
					ctx.URL = &URL{params, r.URL}

					if len(handler.privileges) > 0 {
						var isGranted bool
						if ctx.User != nil {
							for _, pid := range ctx.User.Privileges() {
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
			}
		}
	}
}

func (mux *ApisMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh := w.Header()
	if len(config.CustomHTTPHeaders) > 0 {
		for key, val := range config.CustomHTTPHeaders {
			wh.Set(key, val)
		}
	}
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "webx-server")

	if len(config.HostRedirect) > 0 {
		code := 301 // Permanent redirect, request with GET method
		if r.Method != "GET" {
			// Temporary redirect, request with same method
			// As of Go 1.3, Go does not support status code 308.
			code = 307
		}

		if config.HostRedirect == "force-www" {
			if !strings.HasPrefix(r.Host, "www.") {
				http.Redirect(w, r, path.Join("www."+r.Host, r.URL.String()), code)
				return
			}
		} else if config.HostRedirect == "non-www" {
			if strings.HasPrefix(r.Host, "www.") {
				http.Redirect(w, r, path.Join(strings.TrimPrefix(r.Host, "www."), r.URL.String()), code)
				return
			}
		}
	}

	if mux.router != nil {
		mux.router.ServeHTTP(w, r)
	} else {
		http.Error(w, http.StatusText(404), 404)
	}
}

type AppMux struct {
	app *App
}

func (mux *AppMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.app == nil {
		http.Error(w, "App Not Found", 404)
		return
	}

	// todo: app ssr

	if mux.app.debuging {
		remote, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", mux.app.debugPort))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
		return
	}

	// Serve File
	filePath := utils.CleanPath(path.Join(mux.app.root, r.URL.Path))
Stat:
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 404s will fallback to /index.html
			if topIndexHTML := path.Join(mux.app.root, "index.html"); filePath != topIndexHTML {
				filePath = topIndexHTML
				goto Stat
			}
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if fi.IsDir() {
		filePath = path.Join(filePath, "index.html")
		goto Stat
	}

	// compress text file when the size is greater than 1024 bytes
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		switch strings.ToLower(utils.FileExt(filePath)) {
		case "js", "css", "html", "htm", "xml", "svg", "json", "txt":
			if fi.Size() > 1024 {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Vary", "Accept-Encoding")
				w, err = newGzipResponseWriter(w, gzip.BestSpeed)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				defer w.(*GzipResponseWriter).Close()
			}
		}
	}

	http.ServeFile(w, r, filePath)
}

type GzipResponseWriter struct {
	rawResponseWriter http.ResponseWriter
	gzWriter          io.WriteCloser
}

func newGzipResponseWriter(w http.ResponseWriter, speed int) (grw *GzipResponseWriter, err error) {
	gzipWriter, err := gzip.NewWriterLevel(w, speed)
	if err != nil {
		return
	}

	grw = &GzipResponseWriter{w, gzipWriter}
	return
}

func (w *GzipResponseWriter) Header() http.Header {
	return w.rawResponseWriter.Header()
}

func (w *GzipResponseWriter) WriteHeader(status int) {
	w.rawResponseWriter.WriteHeader(status)
}

func (w *GzipResponseWriter) Write(p []byte) (int, error) {
	return w.gzWriter.Write(p)
}

func (w *GzipResponseWriter) Close() error {
	return w.gzWriter.Close()
}
