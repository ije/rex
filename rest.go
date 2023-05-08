package rex

import (
	"strings"

	"github.com/ije/gox/utils"
)

func match(path string, ctx *Context) bool {
	if strings.HasSuffix(path, "/*") && strings.HasPrefix(ctx.Path.raw, path[:len(path)-1]) {
		return true
	}
	if path == ctx.Path.raw {
		return true
	}
	if strings.ContainsAny(path, ":$?") {
		segments := strings.Split(path[1:], "/")
		if len(segments) == len(ctx.Path.segments) {
			for i, segment := range segments {
				if len(segment) > 0 && (segment[0] == ':' || segment[0] == '$' || segment[0] == '?') {
					continue
				}
				if segment != ctx.Path.segments[i] {
					return false
				}
			}
			return true
		}
	}
	return false
}

func rest(method, path string, handle Handle) Handle {
	if !strings.HasPrefix(path, "/") {
		panic("path must start with '/'")
	}
	path = utils.CleanPath(path)
	return func(ctx *Context) interface{} {
		if ctx.R.Method == method {
			if match(path, ctx) {
				return handle(ctx)
			}
		}
		return nil
	}
}

// HEAD returns a Handle to handle HEAD requests
func HEAD(path string, handle Handle) {
	Use(rest("HEAD", path, handle))
}

// GET returns a Handle to handle GET requests
func GET(path string, handle Handle) {
	Use(rest("GET", path, handle))
}

// POST returns a Handle to handle POST requests
func POST(path string, handle Handle) {
	Use(rest("POST", path, handle))
}

// PUT returns a Handle to handle PUT requests
func PUT(path string, handle Handle) {
	Use(rest("PUT", path, handle))
}

// DELETE returns a Handle to handle DELETE requests
func DELETE(path string, handle Handle) {
	Use(rest("DELETE", path, handle))
}

// PATCH returns a Handle to handle PATCH requests
func PATCH(path string, handle Handle) {
	Use(rest("PATCH", path, handle))
}
