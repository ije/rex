package webx

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ije/gox/utils"
	"github.com/julienschmidt/httprouter"
)

type ApisMux struct {
	router *httprouter.Router
}

func (mux *ApisMux) Register(apis *APIService) {
	if apis == nil {
		return
	}

	if mux.router == nil {
		mux.router = httprouter.New()
		if len(config.AppRoot) > 0 {
			mux.router.NotFound = &AppMux{}
		}
	}

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
			if len(config.AppRoot) > 0 {
				endpoint = path.Join("/api", endpoint)
			}
			routerHandler(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
				ctx := &Context{
					w:   w,
					r:   r,
					URL: &URL{params, r.URL},
				}

				handler.handle(ctx, xs.clone())
			})
		}
	}
}

func (mux *ApisMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh := w.Header()
	if xs.App == nil && len(xs.App.customHTTPHeaders) > 0 {
		for key, val := range xs.App.customHTTPHeaders {
			wh.Set(key, val)
		}
	}
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "webx-server")

	if xs.App != nil {
		code := 301 // Permanent redirect, request with GET method
		if r.Method != "GET" {
			// Temporary redirect, request with same method
			// As of Go 1.3, Go does not support status code 308.
			code = 307
		}

		if xs.App.hostRedirect == "force-www" && !strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, path.Join("www."+r.Host, r.URL.String()), code)
			return
		} else if xs.App.hostRedirect == "non-www" && strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, path.Join(strings.TrimPrefix(r.Host, ".www"), r.URL.String()), code)
			return
		}
	}

	if mux.router != nil {
		mux.router.ServeHTTP(w, r)
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

	// compress file
	if fi.Size() > 1024 && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w, err = newGzipResponseWriter(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer w.(*GzipResponseWriter).Close()
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
