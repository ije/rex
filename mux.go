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

func (mux *ApisMux) Register(apis *APIService) {
	if apis != nil {
		mux.apiss = append(mux.apiss, apis)
	}
}

func (mux *ApisMux) initRouter() {
	mux.router = httprouter.New()

	mux.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if config.Debug {
			http.Error(w, fmt.Sprintf("%v", v), http.StatusInternalServerError)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		if err, ok := v.(error); ok && strings.HasPrefix(err.Error(), "[Context.Session(): ") {
			xs.Log.Error(err)
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
		xs.Log.Error("[panic]", v, buf.String())
	}

	if xs.App != nil {
		apisMux.router.NotFound = &AppMux{}
	}

	for _, apis := range mux.apiss {
		for method, route := range apis.route {
			for endpoint, handler := range route {
				var routerHandler func(string, httprouter.Handle)
				switch method {
				case "OPTIONS":
					routerHandler = mux.router.OPTIONS
				case "HEAD":
					routerHandler = mux.router.HEAD
				case "GET":
					routerHandler = mux.router.GET
				case "POST":
					routerHandler = mux.router.POST
				case "PUT":
					routerHandler = mux.router.PUT
				case "PATCH":
					routerHandler = mux.router.PATCH
				case "DELETE":
					routerHandler = mux.router.DELETE
				}
				if routerHandler == nil {
					continue
				}
				if len(apis.Prefix) > 0 {
					endpoint = path.Join("/"+strings.Trim(apis.Prefix, "/"), endpoint)
				}
				if xs.App != nil {
					endpoint = path.Join("/api", endpoint)
				}
				routerHandler(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
					ctx := &Context{
						URL:            &URL{params, r.URL},
						ResponseWriter: w,
						Request:        r,
					}

					if len(apis.middlewares) > 0 {
						for _, handle := range apis.middlewares {
							handle(ctx, xs.clone())
						}
					}

					if ctx.URL == nil {
						ctx.URL = &URL{params, r.URL}
					}
					if ctx.ResponseWriter == nil {
						ctx.ResponseWriter = w
					}
					if ctx.Request == nil {
						ctx.Request = r
					}

					if len(handler.privileges) > 0 {
						var isGranted bool
						if ctx.User != nil && len(ctx.User.Privileges()) > 0 {
							for _, hp := range handler.privileges {
								for _, up := range ctx.User.Privileges() {
									isGranted = hp.Match(up)
									if isGranted {
										break
									}
								}
								if isGranted {
									break
								}
							}
						}
						if !isGranted {
							ctx.End(http.StatusUnauthorized)
						}
					}

					handler.handle(ctx, xs.clone())
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

		if config.HostRedirect == "force-www" && !strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, path.Join("www."+r.Host, r.URL.String()), code)
			return
		} else if config.HostRedirect == "non-www" && strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, path.Join(strings.TrimPrefix(r.Host, ".www"), r.URL.String()), code)
			return
		}
	}

	if mux.router != nil {
		mux.router.ServeHTTP(w, r)
	} else {
		new(AppMux).ServeHTTP(w, r)
	}
}

type AppMux struct{}

func (mux *AppMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if xs.App == nil {
		http.Error(w, "Missing App", 400)
		return
	}

	// todo: app ssr

	if xs.App.debuging {
		remote, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", xs.App.debugPort))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
		return
	}

	// Serve File
	filePath := utils.CleanPath(path.Join(xs.App.root, r.URL.Path))
Stat:
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 404s will fallback to /index.html
			if topIndexHTML := path.Join(xs.App.root, "index.html"); filePath != topIndexHTML {
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
		case "js", "css", "html", "htm", "xml", "svg", "json", "text":
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
