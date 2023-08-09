package rex

import (
	"time"

	"github.com/ije/rex/session"
)

var defaultRouter = &Router{}
var defaultSessionPool = session.NewMemorySessionPool(time.Hour / 2)
var defaultSessionIdHandler = session.NewCookieIdHandler("")

// Use appends middlewares to current APIS middleware stack.
func Use(middlewares ...Handle) {
	defaultRouter.Use(middlewares...)
}

func AddRoute(method string, pattern string, handle Handle) {
	defaultRouter.AddRoute(method, pattern, handle)
}

// HEAD returns a Handle to handle HEAD requests
func HEAD(pattern string, handles ...Handle) {
	AddRoute("HEAD", pattern, Chain(handles...))
}

// GET returns a Handle to handle GET requests
func GET(pattern string, handles ...Handle) {
	AddRoute("GET", pattern, Chain(handles...))
}

// POST returns a Handle to handle POST requests
func POST(pattern string, handles ...Handle) {
	AddRoute("POST", pattern, Chain(handles...))
}

// PUT returns a Handle to handle PUT requests
func PUT(pattern string, handles ...Handle) {
	AddRoute("PUT", pattern, Chain(handles...))
}

// PATCH returns a Handle to handle PATCH requests
func PATCH(pattern string, handles ...Handle) {
	AddRoute("PATCH", pattern, Chain(handles...))
}

// DELETE returns a Handle to handle DELETE requests
func DELETE(pattern string, handles ...Handle) {
	AddRoute("DELETE", pattern, Chain(handles...))
}
