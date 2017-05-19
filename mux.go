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

type HttpServerMux struct{}

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
	if len(config.CustomHttpHeaders) > 0 {
		for key, val := range config.CustomHttpHeaders {
			wh.Set(key, val)
		}
	}
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "webx-server")

	// filter aliyun slb health check connect
	if r.Method == "HEAD" && r.RequestURI == "/slb-check" {
		w.WriteHeader(200)
		return
	}

	// fix http method
	if m := r.Header.Get("X-Method"); len(m) > 0 {
		switch m = strings.ToUpper(m); m {
		case "HEAD", "GET", "POST", "PUT", "DELETE":
			r.Method = m
		}
	}

	if strings.HasPrefix(r.URL.Path, "/api") {
		wh.Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			wh.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
			wh.Set("Access-Control-Allow-Headers", "Accept,Accept-Encoding,Accept-Lang,Content-Type,Authorization,X-Requested-With,X-Method")
			wh.Set("Access-Control-Allow-Credentials", "true")
			wh.Set("Access-Control-Max-Age", "60")
			return
		}

		endpoint := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api"), "/")
		if len(endpoint) == 0 {
			ctx.End(400)
			return
		}

		handlers, ok := xapis[r.Method]
		if !ok {
			ctx.End(405)
			return
		}

		handler, ok := handlers[endpoint]
		if !ok {
			ctx.End(400)
			return
		}

		if handler.privileges > 0 && (!ctx.Logined() || ctx.LoginedUser().Privileges&handler.privileges == 0) {
			ctx.End(401)
			return
		}

		switch v := handler.handle.(type) {
		case func():
			v()
		case func(*Context):
			v(ctx)
		case func(*XService):
			v(xs.clone())
		case func(*Context, *XService):
			v(ctx, xs.clone())
		case func(*XService, *Context):
			v(xs.clone(), ctx)
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
