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
		if ctx.session != nil {
			xs.Session.PutBack(ctx.session)
		}
		r.Body.Close()
	}()

	wh := w.Header()
	for key, val := range config.CustomHttpHeaders {
		wh.Set(key, val)
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
			ctx.PlainText(400, "Bad Request")
			return
		}

		handlers, ok := xapis[r.Method]
		if !ok {
			ctx.PlainText(405, "Method Not Allowed")
			return
		}

		handler, ok := handlers[endpoint]
		if !ok {
			ctx.PlainText(400, "Bad Request")
			return
		}

		if handler.privileges > 0 && (!ctx.Logined() || ctx.LoginedUser().Privileges&handler.privileges == 0) {
			ctx.PlainText(401, "Unauthorized")
			return
		}

		switch v := handler.handle.(type) {
		case func():
			v()
		case func() []byte:
			ctx.w.Write(v())
		case func() string:
			ctx.w.Write([]byte(v()))
		case func() (int, []byte):
			i, b := v()
			ctx.w.WriteHeader(i)
			ctx.w.Write(b)
		case func() (int, string):
			i, s := v()
			ctx.w.WriteHeader(i)
			ctx.w.Write([]byte(s))
		case func(*Context):
			v(ctx)
		case func(*Context, *XService):
			v(ctx, xs)
		}
		return
	}

	// todo: add/remove `www` in href
	// todo: ssr for seo

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
			ctx.PlainText(404, "File Not Found")
		} else {
			ctx.PlainText(500, "Internal Server Error")
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
		gw := NewGzipResponseWriter(w)
		defer gw.Close()
		w = gw
	}

	http.ServeFile(w, r, filePath)
}

type GzipResponseWriter struct {
	gzWriter          io.WriteCloser
	rawResponseWriter http.ResponseWriter
}

func NewGzipResponseWriter(w http.ResponseWriter) *GzipResponseWriter {
	gzw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	return &GzipResponseWriter{gzw, w}
}

func (w *GzipResponseWriter) Header() http.Header {
	return w.rawResponseWriter.Header()
}

func (w *GzipResponseWriter) Write(p []byte) (int, error) {
	return w.gzWriter.Write(p)
}

func (w *GzipResponseWriter) WriteHeader(status int) {
	w.rawResponseWriter.WriteHeader(status)
}

func (w *GzipResponseWriter) Close() error {
	return w.gzWriter.Close()
}
