package rex

import (
	"html/template"
	"io"
	"os"
	"path"
	"time"
)

// Error defines the error with status.
type Error struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
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

// TypedContent replies to the request with a typed content,
type TypedContent struct {
	Status      int
	Content     []byte
	ContentType string
}

// HTML replies to the request with a html content.
func HTML(content string, status ...int) *TypedContent {
	code := 200
	if len(status) > 0 && status[0] > 0 {
		code = status[0]
	}
	return &TypedContent{code, []byte(content), "text/html; charset=utf-8"}
}

// A Template interface contains the Execute method.
type Template interface {
	Execute(wr io.Writer, data interface{}) error
}

type render struct {
	template Template
	data     interface{}
}

// Render renders the template to the request.
func Render(template Template, data interface{}) interface{} {
	return &render{template, data}
}

// RenderHTML renders the html to the request.
func RenderHTML(html string, data interface{}) interface{} {
	return &render{template.Must(template.New("").Parse(html)), data}
}

type content struct {
	name    string
	motime  time.Time
	content io.ReadSeeker
}

// Content replies to the request using the content in the provided ReadSeeker.
func Content(name string, motime time.Time, r io.ReadSeeker) interface{} {
	return &content{name, motime, r}
}

// File replies to the request using the file content.
func File(name string) interface{} {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return &Error{404, "file not found"}
		}
		return &Error{500, err.Error()}
	}
	if fi.IsDir() {
		return &Error{400, "is a directory"}
	}

	file, err := os.Open(name)
	if err != nil {
		return &Error{500, err.Error()}
	}

	return &content{path.Base(name), fi.ModTime(), file}
}

type fs struct {
	root     string
	fallback string
}

// FS replies to the request with the contents of the file system rooted at root.
func FS(root string, fallback string) interface{} {
	return &fs{root, fallback}
}
