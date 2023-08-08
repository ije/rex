package rex

import (
	"strings"
)

// HEAD returns a Handle to handle HEAD requests
func HEAD(pattern string, handles ...Handle) {
	Use(rest("HEAD", pattern, handles))
}

// GET returns a Handle to handle GET requests
func GET(pattern string, handles ...Handle) {
	Use(rest("GET", pattern, handles))
}

// POST returns a Handle to handle POST requests
func POST(pattern string, handles ...Handle) {
	Use(rest("POST", pattern, handles))
}

// PUT returns a Handle to handle PUT requests
func PUT(pattern string, handles ...Handle) {
	Use(rest("PUT", pattern, handles))
}

// DELETE returns a Handle to handle DELETE requests
func DELETE(pattern string, handles ...Handle) {
	Use(rest("DELETE", pattern, handles))
}

// PATCH returns a Handle to handle PATCH requests
func PATCH(pattern string, handles ...Handle) {
	Use(rest("PATCH", pattern, handles))
}

func rest(method string, pattern string, handles []Handle) Handle {
	if !strings.HasPrefix(pattern, "/") {
		panic("pattern must start with '/'")
	}
	if len(handles) == 0 {
		panic("no handle")
	}
	segments := splitPath(pattern)
	for i, segment := range segments {
		if segment == "*" && i != len(segments)-1 {
			panic("'*' must be the last segment")
		}
	}
	return func(ctx *Context) interface{} {
		if ctx.R.Method == method {
			if match(segments, ctx) {
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

func match(pattern []string, ctx *Context) bool {
	if ctx.Path.raw == "/" && pattern[0] == "" {
		return true
	}
	if reqPath := ctx.Path.segments; len(pattern) <= len(reqPath) {
		for i, p := range pattern {
			if p == "*" {
				ctx.Path.Params["*"] = strings.Join(reqPath[i:], "/")
				return true
			}
			if len(p) > 0 {
				if a := p[0]; a == '$' || a == ':' {
					ctx.Path.Params[p[1:]] = reqPath[i]
					continue
				}
			}
			if p != reqPath[i] {
				return false
			}
		}
		return true
	}
	return false
}
