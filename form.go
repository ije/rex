package rex

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

// A Form to handle http form
type Form struct {
	R *http.Request
}

// Value returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for
// the named component of the request url query.
func (form *Form) Value(key string) string {
	var value string
	switch form.R.Method {
	case "POST", "PUT", "PATCH":
		value = form.R.PostFormValue(key)
	}
	if value == "" {
		value = form.R.FormValue(key)
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

// Require requires a value
func (form *Form) Require(key string) string {
	value := form.Value(key)
	if value == "" {
		panic(&recoverMessage{400, "require form value '" + key + "'"})
	}
	return value
}

// RequireInt requires a value as int
func (form *Form) RequireInt(key string) int64 {
	i, err := form.Int(key)
	if err != nil {
		panic(&recoverMessage{400, "require form value '" + key + "' as int"})
	}
	return i
}

// RequireFloat requires a value as float
func (form *Form) RequireFloat(key string) float64 {
	f, err := form.Float(key)
	if err != nil {
		panic(&recoverMessage{400, "require form value '" + key + "' as float"})
	}
	return f
}

// File returns the first file for the provided form key.
func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}
