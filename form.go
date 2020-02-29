package rex

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

// Form handles http form
type Form struct {
	R *http.Request
}

// Value returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for the named component of the request url query
func (form *Form) Value(key string) string {
	switch form.R.Method {
	case "POST", "PUT", "PATCH":
		return form.R.PostFormValue(key)
	default:
		return form.R.FormValue(key)
	}
}

func (form *Form) Default(key string, defaultValue string) string {
	value := form.Value(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (form *Form) Int(key string) (int64, error) {
	v := strings.TrimSpace(form.Value(key))
	if v == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(v, 10, 64)
}

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
		panic(&contextPanicError{400, "require form value '" + key + "'"})
	}
	return value
}

func (form *Form) RequireInt(key string) int64 {
	i, err := strconv.ParseInt(strings.TrimSpace(form.Require(key)), 10, 64)
	if err != nil {
		panic(&contextPanicError{400, "invalid form value '" + key + "'"})
	}
	return i
}

func (form *Form) RequireFloat(key string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(form.Require(key)), 64)
	if err != nil {
		panic(&contextPanicError{400, "invalid form value '" + key + "'"})
	}
	return f
}

func (form *Form) File(key string) (multipart.File, *multipart.FileHeader, error) {
	return form.R.FormFile(key)
}
