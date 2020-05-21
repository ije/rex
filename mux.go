package rex

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ije/gox/valid"
)

type mux struct {
	forceHTTPS bool
}

func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Connection", "keep-alive")
	header.Set("Server", "rex-serv")

	if m.forceHTTPS && r.TLS == nil && !strings.ContainsRune(r.Host, ':') && !valid.IsIPv4(r.Host) && r.Host != "localhost" {
		code := 301
		if r.Method != "GET" {
			code = 307
		}
		http.Redirect(w, r, fmt.Sprintf("https://%s%s", r.Host, r.RequestURI), code)
		return
	}

	defaultREST.ServeHTTP(w, r)
}
