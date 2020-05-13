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

// Get returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for
// the named component of the request url query.
func (form *Form) Get(key string, defaultValue ...string) string {
	var value string
	switch form.R.Method {
	case "POST", "PUT", "PATCH":
		value = form.R.PostFormValue(key)
	}
	if value == "" {
		value = form.R.FormValue(key)
	}
	if value == "" && len(defaultValue) > 0 {
		value = defaultValue[0]
	}
	return value
}

// GetInt returns the form value as integer
func (form *Form) GetInt(key string) (int64, error) {
	v := strings.TrimSpace(form.Get(key))
	if v == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(v, 10, 64)
}

// GetFloat returns the form value as float
func (form *Form) GetFloat(key string) (float64, error) {
	v := strings.TrimSpace(form.Get(key))
	if v == "" {
		return 0.0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(v, 64)
}

// Require requires a value
func (form *Form) Require(key string) string {
	value := form.Get(key)
	if value == "" {
		panic(&recoverMessage{400, "require form value '" + key + "'"})
	}
	return value
}

// RequireInt requires a value as int
func (form *Form) RequireInt(key string) int64 {
	val := strings.TrimSpace(form.Require(key))
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		panic(&recoverMessage{400, "require form value '" + key + "' as int"})
	}
	return i
}

// RequireFloat requires a value as float
func (form *Form) RequireFloat(key string) float64 {
	val := strings.TrimSpace(form.Require(key))
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		panic(&recoverMessage{400, "require form value '" + key + "' as float"})
	}
	return f
}

// File returns the first file for the provided form key.
func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}
