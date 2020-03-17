package rex

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// Form handles http form
type Form struct {
	R *http.Request
}

// Value returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for
// the named component of the request url query.
func (form *Form) Value(key string) string {
	switch form.R.Method {
	case "POST", "PUT", "PATCH":
		if form.R.PostForm == nil {
			form.R.ParseMultipartForm(defaultMaxMemory)
		}
		if vs := form.R.PostForm[key]; len(vs) > 0 {
			return vs[0]
		}
	}
	return form.R.FormValue(key)
}

// Default returns the defaultValue if the form value is empty
func (form *Form) Default(key string, defaultValue string) string {
	value := form.Value(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Int returns the form value as integer
func (form *Form) Int(key string) (int64, error) {
	v := strings.TrimSpace(form.Value(key))
	if v == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(v, 10, 64)
}

// Float returns the form value as float
func (form *Form) Float(key string) (float64, error) {
	v := strings.TrimSpace(form.Value(key))
	if v == "" {
		return 0.0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(v, 64)
}

func (form *Form) Require(key string) string {
	value := form.Value(key)
	if value == "" {
		panic(&contextPanicError{"require form value '" + key + "'", 400})
	}
	return value
}

func (form *Form) RequireInt(key string) int64 {
	i, err := strconv.ParseInt(strings.TrimSpace(form.Require(key)), 10, 64)
	if err != nil {
		panic(&contextPanicError{"require form value '" + key + "' as int", 400})
	}
	return i
}

func (form *Form) RequireFloat(key string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(form.Require(key)), 64)
	if err != nil {
		panic(&contextPanicError{"require form value '" + key + "' as float", 400})
	}
	return f
}

// File returns the first file for the provided form key.
func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}
