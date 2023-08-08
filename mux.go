package rex

import (
	"fmt"
	"net/http"

	"github.com/ije/gox/valid"
)

type mux struct {
	forceHTTPS bool
}

// ServeHTTP implements http.Handler interface
func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Connection", "keep-alive")
	header.Set("Server", "rex")

	if m.forceHTTPS && r.TLS == nil && r.Host != "localhost" && !valid.IsIPv4(r.Host) {
		code := 301
		if r.Method != "GET" {
			code = 307
		}
		http.Redirect(w, r, fmt.Sprintf("https://%s%s", r.Host, r.RequestURI), code)
		return
	}

	defaultRouter.ServeHTTP(w, r)
}
