package rex

import (
	"fmt"
	"net/http"
)

type mux struct {
	forceHTTPS bool
}

func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh := w.Header()
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "rex-serv")

	if m.forceHTTPS && r.TLS == nil {
		code := 301
		if r.Method != "GET" {
			code = 307
		}
		http.Redirect(w, r, fmt.Sprintf("https://%s/%s", r.Host, r.RequestURI), code)
		return
	}

	defaultRouter.ServeHTTP(w, r)
}
