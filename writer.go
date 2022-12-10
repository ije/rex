package rex

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

// A responseWriter is used by rex.Context to construct a HTTP response.
type responseWriter struct {
	status      int
	written     int
	headerSent  bool
	httpWriter  http.ResponseWriter
	compression io.WriteCloser
}

// Hijack lets the caller take over the connection.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.httpWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("the raw response writer does not implement the http.Hijacker")
}

// Header returns the header map that will be sent by WriteHeader.
func (w *responseWriter) Header() http.Header {
	return w.httpWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *responseWriter) WriteHeader(status int) {
	if !w.headerSent {
		w.status = status
		w.httpWriter.WriteHeader(status)
		w.headerSent = true
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *responseWriter) Write(p []byte) (n int, err error) {
	if !w.headerSent {
		w.headerSent = true
	}
	var wr io.Writer = w.httpWriter
	if w.compression != nil {
		wr = w.compression
	}
	n, err = wr.Write(p)
	if n > 0 {
		w.written += n
	}
	return
}

func (w *responseWriter) Close() error {
	if w.compression != nil {
		return w.compression.Close()
	}
	return nil
}
