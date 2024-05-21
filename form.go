package rex

import (
	"fmt"
	"mime/multipart"
	"net/http"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// A Form to handle request form data.
type Form struct {
	R *http.Request
}

// Has checks the value for the key whether exists.
func (form *Form) Has(key string) bool {
	r := form.R
	if m := r.Method; m == "POST" || m == "PUT" || m == "PATCH" {
		if r.PostForm == nil {
			r.ParseMultipartForm(defaultMaxMemory)
		}
		_, ok := r.PostForm[key]
		if ok {
			return true
		}
	}
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	_, ok := r.Form[key]
	return ok
}

// Value returns the value for the key.
func (form *Form) Value(key string) string {
	r := form.R
	if m := r.Method; m == "POST" || m == "PUT" || m == "PATCH" {
		if r.PostForm == nil {
			r.ParseMultipartForm(defaultMaxMemory)
		}
		if vs, ok := r.PostForm[key]; ok {
			if len(vs) > 0 {
				return vs[0]
			}
			return ""
		}
	}
	return r.FormValue(key)
}

// File returns the first file for the provided form key.
func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}

// Require returns the value for the key, if the value is empty, it will panic.
func (form *Form) Require(key string) string {
	if !form.Has(key) {
		panic(&invalid{400, fmt.Sprintf("require form field '%s'", key)})
	}
	return form.Value(key)
}
