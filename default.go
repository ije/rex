package rex

import (
	"time"

	"github.com/ije/rex/session"
)

var defaultHanlder = &Handler{}
var defaultSessionPool = session.NewMemorySessionPool(time.Hour / 2)
var defaultSIDStore = session.NewCookieSIDStore("")

// Default returns the default REST
func Default() *Handler {
	return defaultHanlder
}

// Prefix adds prefix for each api path, like "v2"
func Prefix(prefix string) *Handler {
	return defaultHanlder.Prefix(prefix)
}

// Use appends middlewares to current APIS middleware stack.
func Use(middlewares ...Handle) {
	defaultHanlder.Use(middlewares...)
}

// Query adds a query api
func Query(endpoint string, handles ...Handle) {
	defaultHanlder.Query(endpoint, handles...)
}

// Mutation adds a mutation api
func Mutation(endpoint string, handles ...Handle) {
	defaultHanlder.Mutation(endpoint, handles...)
}
