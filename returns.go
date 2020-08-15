package rex

import (
	"html/template"
	"io"
	"os"
	"path"
	"time"
)

type HTTPError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func Error(message string, status int) interface{} {
	return &HTTPError{status, message}
}

type redirect struct {
	status int
	url    string
}

// Redirect replies to the request with a redirect to url,
// which may be a path relative to the request path.
func Redirect(url string, status int) interface{} {
	return &redirect{status, url}
}

type htm struct {
	status int
	html   string
}

func HTML(html string, status int) interface{} {
	return &htm{status, html}
}

// A Template interface contains the Execute method.
type Template interface {
	Execute(wr io.Writer, data interface{}) error
}

type render struct {
	template Template
	data     interface{}
}

func Render(t Template, data interface{}) interface{} {
	return &render{t, data}
}

func RenderHTML(html string, data interface{}) interface{} {
	return &render{template.Must(template.New("").Parse(html)), data}
}

type content struct {
	name    string
	motime  time.Time
	content io.ReadSeeker
}

func Content(name string, motime time.Time, r io.ReadSeeker) interface{} {
	return &content{name, motime, r}
}

func File(name string) interface{} {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return Error("file not found", 404)
		}
		return Error(err.Error(), 500)
	}
	if fi.IsDir() {
		return Error("is a directory", 400)
	}

	file, err := os.Open(name)
	if err != nil {
		return Error(err.Error(), 500)
	}

	return &content{path.Base(name), fi.ModTime(), file}
}

type static struct {
	root     string
	fallback string
}

func Static(root string, fallback string) interface{} {
	return &static{root, fallback}
}
