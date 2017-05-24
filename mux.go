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
)

type HttpServerMux struct {
	CustomHttpHeaders map[string]string
	PlainAPIServer    bool
}

func (mux *HttpServerMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &Context{w: w, r: r, host: r.Host}

	defer func() {
		if v := recover(); v != nil {
			var (
				j    int
				pc   uintptr
				file string
				line int
				ok   bool
			)
			i := 2
			buf := bytes.NewBuffer(nil)
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
			ctx.Error(errf(buf.String()))
		}
		r.Body.Close()
	}()

	wh := w.Header()
	if len(mux.CustomHttpHeaders) > 0 {
		for key, val := range mux.CustomHttpHeaders {
			wh.Set(key, val)
		}
	}
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "webx-server")

	// filter aliyun slb health check connect
	if r.Method == "HEAD" && r.RequestURI == "/slb-check" {
		w.WriteHeader(204)
		return
	}

	// fix http method
	if m := r.Header.Get("X-Method"); len(m) > 0 {
		switch m = strings.ToUpper(m); m {
		case "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE":
			r.Method = m
		}
	}

	if mux.PlainAPIServer || strings.HasPrefix(r.URL.Path, "/api") {
		var handlers map[string]apiHandler
		var ok bool
		var prefix string
		for _, apis := range xapis {
			handlers, ok = apis[r.Method]
			if ok {
				prefix = apis.getConfig("prefix")
				break
			}
		}
		if !ok {
			ctx.End(405)
			return
		}

		var endpoint string
		if mux.PlainAPIServer {
			endpoint = r.URL.Path
		} else {
			endpoint = strings.TrimPrefix(r.URL.Path, "/api")
		}
		endpoint = strings.Trim(endpoint, "/")
		if len(prefix) > 0 {
			endpoint = strings.TrimPrefix(endpoint, strf("%s/", strings.Trim(prefix, "/")))
		}
		if len(endpoint) == 0 {
			ctx.End(400)
			return
		}

		handler, ok := handlers[endpoint]
		if !ok {
			handler, ok = handlers["*"]
		}
		if !ok {
			ctx.End(400)
			return
		}

		if r.Method == "options" {
			switch v := handler.handle.(type) {
			case func() *CORS:
				cors := v()
				if cors == nil {
					ctx.End(400)
					return
				}

				wh.Set("Access-Control-Allow-Origin", cors.Origin)
				wh.Set("Access-Control-Allow-Methods", cors.Methods)
				wh.Set("Access-Control-Allow-Headers", cors.Headers)
				if cors.Credentials {
					wh.Set("Access-Control-Allow-Credentials", "true")
				}
				if cors.MaxAge > 0 {
					wh.Set("Access-Control-Max-Age", strf("%d", cors.MaxAge))
				}
				w.WriteHeader(204)
			}
		}

		if handler.privileges > 0 && (!ctx.Logined() || ctx.LoginedUser().Privileges&handler.privileges == 0) {
			ctx.End(401)
			return
		}

		switch v := handler.handle.(type) {
		case func():
			v()
			w.WriteHeader(204)
		case func(*XService):
			v(xs.clone())
			w.WriteHeader(204)
		case func(*Context):
			v(ctx)
		case func(*Context, *XService):
			v(ctx, xs.clone())
		case func(*XService, *Context):
			v(xs.clone(), ctx)
		default:
			ctx.End(400)
		}
		return
	}

	// todo: add/remove `www` in href
	// todo: ssr for seo

	if xs.App == nil {
		ctx.End(400, "Missing App")
		return
	}

	if xs.App.debuging {
		remote, err := url.Parse(strf("http://127.0.0.1:%d", debugPort))
		if err != nil {
			ctx.Error(err)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
		return
	}

	// Serve File
	filePath := utils.CleanPath(path.Join(xs.App.root, r.URL.Path), false)
Stat:
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if topIndexHtml := path.Join(xs.App.root, "index.html"); filePath != topIndexHtml {
				filePath = topIndexHtml
				goto Stat
			}
			ctx.End(404)
		} else {
			ctx.End(500)
		}
		return
	}

	if fi.IsDir() {
		filePath = path.Join(filePath, "index.html")
		goto Stat
	}

	if fi.Size() > 1024 && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		wh.Set("Content-Encoding", "gzip")
		wh.Set("Vary", "Accept-Encoding")
		gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			ctx.End(500)
			return
		}
		defer gzw.Close()
		w = &gzipResponseWriter{w, gzw}
	}

	http.ServeFile(w, r, filePath)
}

type gzipResponseWriter struct {
	rawResponseWriter http.ResponseWriter
	gzWriter          io.WriteCloser
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.rawResponseWriter.Header()
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.rawResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.gzWriter.Write(p)
}
