package rex

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ije/gox/utils"
)

type mux struct {
	rests      map[string][][]*REST
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

	host, _ := utils.SplitByLastByte(r.Host, ':')
	prefixs, ok := m.rests[host]
	if !ok && strings.HasPrefix(host, "www.") {
		prefixs, ok = m.rests[strings.TrimPrefix(host, "www.")]
	}
	if !ok {
		prefixs, ok = m.rests["*"]
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(prefixs) > 0 {
		for _, rests := range prefixs {
			if rests[0].prefix != "" && strings.HasPrefix(r.URL.Path, "/"+rests[0].prefix) {
				rests[0].ServeHTTP(w, r)
				return
			}
		}
		if rests := prefixs[len(prefixs)-1]; rests[0].prefix == "" {
			rests[0].ServeHTTP(w, r)
			return
		}
	}

	http.NotFound(w, r)
}
