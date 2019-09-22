package rex

import (
	"net/http"
)

var gRESTs = map[string][][]*REST{}

func global(rest *REST) {
	// clean up
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

func linkRESTs() map[string][][]*REST {
	_gRESTs := map[string][][]*REST{}
	for host, prefixs := range gRESTs {
		var _prefixs [][]*REST
		for _, rests := range prefixs {
			var _rests []*REST
			for _, rest := range rests {
				if rest.router != nil {
					_rests = append(_rests, rest)
				}
			}
			if len(_rests) > 0 {
				_prefixs = append(_prefixs, _rests)
			}
		}
		if len(_prefixs) > 0 {
			_gRESTs[host] = _prefixs
		}
	}

	for _, prefixs := range _gRESTs {
		for _, rests := range prefixs {
			if len(rests) > 1 {
				for index, rest := range rests {
					func(index int, rest *REST, rests []*REST) {
						rest.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
							if index+1 <= len(rests)-1 {
								rests[index+1].ServeHTTP(w, r)
								return
							}
							if f := rests[0]; f.notFoundHandle != nil {
								f.serve(w, r, nil, f.notFoundHandle)
							} else if rest.notFoundHandle != nil {
								rest.serve(w, r, nil, rest.notFoundHandle)
							} else {
								rest.serve(w, r, nil, func(ctx *Context) {
									ctx.End(404)
								})
							}
						})
					}(index, rest, rests)
				}
			}
		}
	}

	return _gRESTs
}
