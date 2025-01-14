package rex

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

// A Writer implements the http.ResponseWriter interface.
type rexWriter struct {
	ctx          *Context
	code         int
	isHeaderSent bool
	writeN       int
	rawWriter    http.ResponseWriter
	zWriter      io.WriteCloser
}

// Hijack lets the caller take over the connection.
func (w *rexWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.rawWriter.(http.Hijacker)
	if ok {
		return h.Hijack()
	}

	return nil, nil, errors.New("the raw response writer does not implement the http.Hijacker")
}

// Flush sends any buffered data to the client.
func (w *rexWriter) Flush() {
	f, ok := w.rawWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *rexWriter) Header() http.Header {
	return w.rawWriter.Header()
}

// WriteHeader sends a HTTP response header with the provided status code.
func (w *rexWriter) WriteHeader(code int) {
	if !w.isHeaderSent {
		w.rawWriter.WriteHeader(code)
		w.code = code
		w.isHeaderSent = true
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *rexWriter) Write(p []byte) (n int, err error) {
	if !w.isHeaderSent {
		w.isHeaderSent = true
	}
	var wr io.Writer = w.rawWriter
	if w.zWriter != nil {
		wr = w.zWriter
	}
	n, err = wr.Write(p)
	if n > 0 {
		w.writeN += n
	}
	return
}

// Close closes the underlying connection.
func (w *rexWriter) Close() error {
	if w.zWriter != nil {
		return w.zWriter.Close()
	}
	return nil
}
