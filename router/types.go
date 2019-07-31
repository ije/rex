package router

import (
	"net/http"
)

type node struct {
	name           string
	staticChildren []*node
	paramChild     *node
	catchAllChild  *node
	validate       string
	handle         Handle
}

func (n *node) lookup(name string) (*node, bool) {
	for _, nod := range n.staticChildren {
		if nod.name == name {
			return nod, true
		}
	}
	return nil, false
}

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}

// Handle is a function that can be registered to a route to handle HTTP requests.
type Handle func(w http.ResponseWriter, r *http.Request, params Params)

// ValidateFn is a function to validate route params
type ValidateFn func(s string) (ok bool)
