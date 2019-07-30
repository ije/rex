package rex

var defaultREST = New("*", "")

// Use appends middleware to the REST middleware stack.
func Use(middlewares ...Handle) {
	defaultREST.Use(middlewares...)
}

// NotFound handles the requests that are not routed
func NotFound(handles ...Handle) {
	defaultREST.NotFound(handles...)
}

// Options is a shortcut for router.Handle("OPTIONS", path, handles)
func Options(path string, handles ...Handle) {
	defaultREST.Options(path, handles...)
}

// Head is a shortcut for router.Handle("HEAD", path, handles)
func Head(path string, handles ...Handle) {
	defaultREST.Head(path, handles...)
}

// Get is a shortcut for router.Handle("GET", path, handles)
func Get(path string, handles ...Handle) {
	defaultREST.Get(path, handles...)
}

// Post is a shortcut for router.Handle("POST", path, handles)
func Post(path string, handles ...Handle) {
	defaultREST.Post(path, handles...)
}

// Put is a shortcut for router.Handle("PUT", path, handles)
func Put(path string, handles ...Handle) {
	defaultREST.Put(path, handles...)
}

// Patch is a shortcut for router.Handle("PATCH", path, handles)
func Patch(path string, handles ...Handle) {
	defaultREST.Patch(path, handles...)
}

// Delete is a shortcut for router.Handle("DELETE", path, handles)
func Delete(path string, handles ...Handle) {
	defaultREST.Delete(path, handles...)
}
