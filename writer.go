package rex

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

// A Writer implements the http.ResponseWriter interface.
type rexWriter struct {
	ctx         *Context
	status      int
	written     int
	headerSent  bool
	httpWriter  http.ResponseWriter
	compression io.WriteCloser
}

// Hijack lets the caller take over the connection.
func (w *rexWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.httpWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("the raw response writer does not implement the http.Hijacker")
}

// Flush sends any buffered data to the client.
func (w *rexWriter) Flush() {
	f, ok := w.httpWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *rexWriter) Header() http.Header {
	return w.httpWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *rexWriter) WriteHeader(status int) {
	if !w.headerSent {
		w.status = status
		w.httpWriter.WriteHeader(status)
		w.headerSent = true
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *rexWriter) Write(p []byte) (n int, err error) {
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

// Close closes the underlying connection.
func (w *rexWriter) Close() error {
	if w.compression != nil {
		return w.compression.Close()
	}
	return nil
}
