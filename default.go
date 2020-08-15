package rex

import (
	"time"

	"github.com/ije/rex/session"
)

var defaultAPIHanlder = &APIHandler{}
var defaultSessionPool = session.NewMemorySessionPool(time.Hour / 2)
var defaultSIDStore = &session.CookieSIDStore{}

// Default returns the default REST
func Default() *APIHandler {
	return defaultAPIHanlder
}

// Use appends middlewares to current APIS middleware stack.
func Use(middlewares ...Handle) {
	defaultAPIHanlder.Use(middlewares...)
}

// Query adds a query api
func Query(endpoint string, handles ...Handle) {
	defaultAPIHanlder.Query(endpoint, handles...)
}

// Mutation adds a mutation api
func Mutation(endpoint string, handles ...Handle) {
	defaultAPIHanlder.Mutation(endpoint, handles...)
}
