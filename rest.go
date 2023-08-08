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

func rest(method string, path string, handles []Handle) Handle {
	if !strings.HasPrefix(path, "/") {
		panic("path must start with '/'")
	}
	if len(handles) == 0 {
		panic("handles must not be empty")
	}
	path = utils.CleanPath(path)
	return func(ctx *Context) interface{} {
		if ctx.R.Method == method {
			if match(path, ctx) {
				w, r, path, form, store := ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store
				for _, handle := range handles {
					v := handle(ctx)
					if v != nil {
						return v
					}
					ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = w, r, path, form, store
				}
			}
		}
		return nil
	}
}

// HEAD returns a Handle to handle HEAD requests
func HEAD(path string, handles ...Handle) {
	Use(rest("HEAD", path, handles))
}

// GET returns a Handle to handle GET requests
func GET(path string, handles ...Handle) {
	Use(rest("GET", path, handles))
}

// POST returns a Handle to handle POST requests
func POST(path string, handles ...Handle) {
	Use(rest("POST", path, handles))
}

// PUT returns a Handle to handle PUT requests
func PUT(path string, handles ...Handle) {
	Use(rest("PUT", path, handles))
}

// DELETE returns a Handle to handle DELETE requests
func DELETE(path string, handles ...Handle) {
	Use(rest("DELETE", path, handles))
}

// PATCH returns a Handle to handle PATCH requests
func PATCH(path string, handles ...Handle) {
	Use(rest("PATCH", path, handles))
}
