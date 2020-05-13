package rex

import (
	"bufio"
	"bytes"
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
	buffer     *bytes.Buffer
	written    int
	status     int
	gzipWriter io.WriteCloser
	rawWriter  http.ResponseWriter
}

func newGzipWriter(w http.ResponseWriter) (gzw *gzipResponseWriter) {
	gzw = &gzipResponseWriter{bytes.NewBuffer(nil), 0, 200, nil, w}
	return
}

// Header returns the header map that will be sent by WriteHeader.
func (w *gzipResponseWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *gzipResponseWriter) WriteHeader(status int) {
	w.status = status
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if w.written >= 1024 {
		return w.gzipWriter.Write(p)
	}

	n, err := w.buffer.Write(p)
	if err != nil {
		return n, err
	}

	w.written += n
	if w.written >= 1024 {
		// w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		w.rawWriter.WriteHeader(w.status)
		w.gzipWriter, _ = gzip.NewWriterLevel(w.rawWriter, gzip.BestSpeed)
		w.gzipWriter.Write(w.buffer.Bytes())
		w.buffer.Reset()
	}
	return n, err
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
	if w.gzipWriter != nil {
		return w.gzipWriter.Close()
	} else if w.buffer.Len() > 0 {
		w.rawWriter.WriteHeader(w.status)
		_, err := w.rawWriter.Write(w.buffer.Bytes())
		return err
	}
	return nil
}
