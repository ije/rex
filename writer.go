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
	ctx        *Context
	code       int
	header     http.Header
	headerSent bool
	writeN     int
	httpWriter http.ResponseWriter
	compWriter io.WriteCloser
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
	return w.header
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *rexWriter) WriteHeader(code int) {
	if w.headerSent {
		return
	}
	w.code = code
	w.httpWriter.WriteHeader(code)
	w.headerSent = true
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *rexWriter) Write(p []byte) (n int, err error) {
	if !w.headerSent {
		w.headerSent = true
	}
	var wr io.Writer = w.httpWriter
	if w.compWriter != nil {
		wr = w.compWriter
	}
	n, err = wr.Write(p)
	if n > 0 {
		w.writeN += n
	}
	return
}

// Close closes the underlying connection.
func (w *rexWriter) Close() error {
	if w.compWriter != nil {
		return w.compWriter.Close()
	}
	return nil
}
