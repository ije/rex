package rex

import (
	"net/url"

	"github.com/julienschmidt/httprouter"
)

// A URL is a *url.URL with httprouter.Params
type URL struct {
	Params httprouter.Params
	*url.URL
}

// Param returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (url *URL) Param(name string) string {
	return url.Params.ByName(name)
}