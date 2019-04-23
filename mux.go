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

	"github.com/ije/gox/utils"
	"github.com/julienschmidt/httprouter"
)

// Mux is a HTTP request multiplexer.
// The inner router provides by github.com/julienschmidt/httprouter.
type Mux struct {
	Config
	router *httprouter.Router
}

func (mux *Mux) initRouter() *httprouter.Router {
	router := httprouter.New()
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if mux.Debug {
			http.Error(w, fmt.Sprintf("%v", v), 500)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}

		if err, ok := v.(*ctxPanicError); ok {
			if mux.Logger != nil {
				mux.Logger.Errorf(err.msg)
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
		router.NotFound = &staticMux{utils.CleanPath(mux.Root), mux.NotFoundHandler}
	} else if mux.NotFoundHandler != nil {
		router.NotFound = mux.NotFoundHandler
	}
	return router
}

// RegisterREST registers a REST instacce
func (mux *Mux) RegisterREST(rest *REST) {
	if rest == nil {
		return
	}

	if mux.router == nil {
		mux.router = mux.initRouter()
	}

	for method, route := range rest.route {
		for endpoint, handles := range route {
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
			if len(rest.Prefix) > 0 {
				endpoint = path.Join("/"+strings.Trim(rest.Prefix, "/"), endpoint)
			}
			func(mux *Mux, routerHandle func(string, httprouter.Handle), endpoint string, handles []RESTHandle, rest *REST) {
				routerHandle(endpoint, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
					url := &URL{params, r.URL}
					state := NewState()
					ctx := &Context{
						W:           w,
						R:           r,
						URL:         url,
						State:       state,
						handles:     append(rest.middlewares, handles...),
						handleIndex: -1,
						privileges:  map[string]struct{}{},
						mux:         mux,
					}
					ctx.Next()
				})
			}(mux, routerHandle, endpoint, handles, rest)
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
				mux.AccessLogger.Log(
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

	if len(mux.HostRedirectRule) > 0 && r.Method == "GET" {
		proto := "http"
		if r.TLS != nil {
			proto = "https"
		}
		if strings.Contains(mux.HostRedirectRule, "https") {
			if proto == "http" {
				proto = "https"
			}
		}
		if strings.Contains(mux.HostRedirectRule, "www") {
			if !strings.HasPrefix(r.Host, "www.") {
				http.Redirect(w, r, proto+"://"+path.Join("www."+r.Host, r.URL.String()), http.StatusMovedPermanently)
				return
			}
		} else if strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, proto+"://"+path.Join(strings.TrimPrefix(r.Host, "www."), r.URL.String()), http.StatusMovedPermanently)
			return
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
	rootIndexHTML := path.Join(mux.root, "index.html")
	file := path.Join(mux.root, utils.CleanPath(r.URL.Path))
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
		for _, ext := range []string{"js", "js.map", "json", "css", "html", "htm", "xml", "svg", "txt"} {
			if strings.HasSuffix(strings.ToLower(file), "."+ext) {
				if fi.Size() > 1024 {
					if w, ok := w.(*ResponseWriter); ok {
						gzw := newGzResponseWriter(w.rawWriter)
						defer gzw.Close()
						w.rawWriter = gzw
					}
				}
			}
		}
	}

	http.ServeFile(w, r, file)
}

// A ResponseWriter is used by an rex mux to construct an HTTP response.
type ResponseWriter struct {
	status      int
	writedBytes int
	rawWriter   http.ResponseWriter
}

// Header returns the header map that will be sent by WriteHeader.
func (w *ResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *ResponseWriter) WriteHeader(status int) {
	w.status = status
	if w.writedBytes == 0 {
		w.rawWriter.WriteHeader(status)
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	n, err = w.rawWriter.Write(p)
	w.writedBytes += n
	return
}

// Hijack lets the caller take over the connection.
func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.rawWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("The raw response writer does not implement the http.Hijacker")
}

// A gzResponseWriter is used by an rex mux to construct an HTTP response with gzip compress.
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
