package rex

import (
	"time"

	"github.com/ije/rex/session"
)

var defaultHanlder = &Handler{}
var defaultSessionPool = session.NewMemorySessionPool(time.Hour / 2)
var defaultSessionIdHandler = session.NewCookieIdHandler("")

// Default returns the default REST
func Default() *Handler {
	return defaultHanlder
}

// Use appends middlewares to current APIS middleware stack.
func Use(middlewares ...Handle) {
	defaultHanlder.Use(middlewares...)
}
