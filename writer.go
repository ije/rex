package rex

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
)

// A responseWriter is used by rex.Context to construct a HTTP response.
type responseWriter struct {
	status    int
	written   int
	rawWriter http.ResponseWriter
}

// Header returns the header map that will be sent by WriteHeader.
func (w *responseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	if w.written == 0 {
		w.rawWriter.WriteHeader(status)
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *responseWriter) Write(p []byte) (n int, err error) {
	n, err = w.rawWriter.Write(p)
	w.written += n
	return
}

// Hijack lets the caller take over the connection.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
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

func newGzipWriter(w http.ResponseWriter) *gzipResponseWriter {
	// w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("Content-Encoding", "gzip")
	gzipWriter, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	return &gzipResponseWriter{gzipWriter, w}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *gzipResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *gzipResponseWriter) WriteHeader(status int) {
	w.rawWriter.WriteHeader(status)
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.gzipWriter.Write(p)
}

// Hijack lets the caller take over the connection.
func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.rawWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("The raw response writer does not implement the http.Hijacker")
}

func (w *gzipResponseWriter) Close() error {
	return w.gzipWriter.Close()
}
