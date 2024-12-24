package rex

import (
	"log"
	"time"

	"github.com/ije/rex/session"
)

const compressMinSize = 1024

var defaultMux = &Mux{}
var defaultSessionPool = session.NewMemorySessionPool(time.Hour / 2)
var defaultSessionIdHandler = session.NewCookieSidHandler("SID")
var defaultLogger = &log.Logger{}

// Use appends middlewares to current APIS middleware stack.
func Use(middlewares ...Handle) {
	defaultMux.Use(middlewares...)
}

// AddRoute adds a route.
func AddRoute(pattern string, handle Handle) {
	defaultMux.AddRoute(pattern, handle)
}

// HEAD returns a Handle to handle HEAD requests
func HEAD(pattern string, handles ...Handle) {
	AddRoute("HEAD "+pattern, Chain(handles...))
}

// GET returns a Handle to handle GET requests
func GET(pattern string, handles ...Handle) {
	AddRoute("GET "+pattern, Chain(handles...))
}

// POST returns a Handle to handle POST requests
func POST(pattern string, handles ...Handle) {
	AddRoute("POST "+pattern, Chain(handles...))
}

// PUT returns a Handle to handle PUT requests
func PUT(pattern string, handles ...Handle) {
	AddRoute("PUT "+pattern, Chain(handles...))
}

// PATCH returns a Handle to handle PATCH requests
func PATCH(pattern string, handles ...Handle) {
	AddRoute("PATCH "+pattern, Chain(handles...))
}

// DELETE returns a Handle to handle DELETE requests
func DELETE(pattern string, handles ...Handle) {
	AddRoute("DELETE "+pattern, Chain(handles...))
}
