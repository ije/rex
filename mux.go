package rex

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ije/gox/utils"
)

type mux struct {
	rests  map[string][][]*REST
	config Config
}

func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh := w.Header()
	wh.Set("Connection", "keep-alive")
	wh.Set("Server", "rex-serv")

	if m.config.TLS.AutoRedirect && r.TLS == nil {
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

var gRESTs = map[string][][]*REST{}

func global(rest *REST) {
	// clean
	for host, prefixs := range gRESTs {
		var _prefixs [][]*REST
		for _, rests := range prefixs {
			var _rests []*REST
			for _, _rest := range rests {
				if _rest != rest {
					_rests = append(_rests, _rest)
				}
			}
			if len(_rests) > 0 {
				_prefixs = append(_prefixs, _rests)
			}
		}
		if len(_prefixs) > 0 {
			gRESTs[host] = _prefixs
		}
	}

	// append or insert
	prefixs, ok := gRESTs[rest.host]
	if ok {
		for i, rests := range prefixs {
			if rest.prefix == rests[0].prefix {
				prefixs[i] = append(rests, rest)
				return
			}
		}
	}
	if len(prefixs) == 0 {
		prefixs = [][]*REST{[]*REST{rest}}
	} else {
		insertIndex := 0
		for i, rests := range prefixs {
			if len(rest.prefix) > len(rests[0].prefix) {
				insertIndex = i
				break
			}
		}
		tmp := make([][]*REST, len(prefixs)+1)
		copy(tmp, prefixs[:insertIndex])
		copy(tmp[insertIndex+1:], prefixs[insertIndex:])
		tmp[insertIndex] = []*REST{rest}
		prefixs = tmp
	}
	gRESTs[rest.host] = prefixs
}
