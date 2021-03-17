package rex

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// A Form to handle request form data.
type Form struct {
	R *http.Request
}

// IsNil checks the value for the key whether is nil.
func (form *Form) IsNil(key string) bool {
	if form.R.Method == "POST" {
		if form.R.PostForm == nil {
			form.R.ParseMultipartForm(defaultMaxMemory)
		}
		_, ok := form.R.PostForm[key]
		if ok {
			return false
		}
	}
	if form.R.Form == nil {
		form.R.ParseMultipartForm(defaultMaxMemory)
	}
	_, ok := form.R.Form[key]
	return !ok
}

// Value returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for
// the named component of the request url query.
func (form *Form) Value(key string) string {
	var value string
	if form.R.Method == "POST" {
		value = form.R.PostFormValue(key)
	}
	if value == "" {
		value = form.R.FormValue(key)
	}
	return value
}

// Int returns the form value as integer
func (form *Form) Int(key string) (int64, error) {
	value := form.Value(key)
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(value, 10, 64)
}

// Float returns the form value as float
func (form *Form) Float(key string) (float64, error) {
	value := form.Value(key)
	if value == "" {
		return 0.0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(value, 64)
}

// Require requires a value
func (form *Form) Require(key string) string {
	value := form.Value(key)
	if value == "" {
		panic(&recoverError{400, fmt.Sprintf("require form value '%s'", key)})
	}
	return value
}

// RequireInt requires a value as int
func (form *Form) RequireInt(key string) int64 {
	i, err := form.Int(key)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require form value '%s' as int", key)})
	}
	return i
}

// RequireFloat requires a value as float
func (form *Form) RequireFloat(key string) float64 {
	f, err := form.Float(key)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require form value '%s' as float", key)})
	}
	return f
}

// File returns the first file for the provided form key.
func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}
