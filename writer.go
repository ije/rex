package rex

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
)

// A clearResponseWriter is used by rex.Context to construct a HTTP response.
type clearResponseWriter struct {
	status    int
	writed    int
	rawWriter http.ResponseWriter
}

// Header returns the header map that will be sent by WriteHeader.
func (w *clearResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *clearResponseWriter) WriteHeader(status int) {
	w.status = status
	if w.writed == 0 {
		w.rawWriter.WriteHeader(status)
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *clearResponseWriter) Write(p []byte) (n int, err error) {
	n, err = w.rawWriter.Write(p)
	w.writed += n
	return
}

// Hijack lets the caller take over the connection.
func (w *clearResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.rawWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("The raw response writer does not implement the http.Hijacker")
}

// A gzipResponseWriter is used by rex.Context to construct a HTTP response with gzip compress.
type gzipResponseWriter struct {
	gzipWriter io.WriteCloser
	rawWriter  http.ResponseWriter
}

func newGzipWriter(w http.ResponseWriter) (gzw *gzipResponseWriter) {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	gzipWriter, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	gzw = &gzipResponseWriter{gzipWriter, w}
	return
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.rawWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.gzipWriter.Write(p)
}

func (w *gzipResponseWriter) Close() error {
	return w.gzipWriter.Close()
}
