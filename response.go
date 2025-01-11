package rex

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
)

type invalid struct {
	code    int
	message string
}

// Invalid returns an invalid error with code and message.
func Invalid(code int, v ...string) any {
	var messsage string
	if len(v) > 0 {
		messsage = strings.Join(v, " ")
	} else {
		messsage = http.StatusText(code)
	}
	return &invalid{code, messsage}
}

// Error defines an error with code and message.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// Err returns an error with code and message.
func Err(code int, v ...string) any {
	if code < 400 || code >= 600 {
		panic("invalid status code")
	}
	var messsage string
	if len(v) > 0 {
		messsage = strings.Join(v, " ")
	} else {
		messsage = http.StatusText(code)
	}
	return &Error{
		Code:    code,
		Message: messsage,
	}
}

type redirect struct {
	status int
	url    string
}

// Redirect replies to the request with a redirect to url,
// which may be a path relative to the request path.
func Redirect(url string, status int) any {
	if url == "" {
		url = "/"
	}
	if status < 300 || status >= 400 {
		status = 302
	}
	return &redirect{status, url}
}

type status struct {
	code    int
	content any
}

// Status replies to the request using the payload in the status.
func Status(code int, content any) any {
	return &status{code, content}
}

// Template is an interface for template.
type Template interface {
	Name() string
	Execute(wr io.Writer, data any) error
}

func Tpl(text string) Template {
	return template.Must(template.New("index.html").Parse(text))
}

// Render renders the template with the given data.
func Render(t Template, data any) any {
	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, data); err != nil {
		panic(&invalid{500, err.Error()})
	}

	return &content{
		name:    t.Name(),
		content: bytes.NewReader(buf.Bytes()),
	}
}

type noContent struct{}

// NoContent replies to the request with no content.
func NoContent() any {
	return &noContent{}
}

type content struct {
	name    string
	mtime   time.Time
	content io.Reader
}

// Content replies to the request using the content in the provided Reader.
func Content(name string, mtime time.Time, r io.Reader) any {
	return &content{name, mtime, r}
}

// HTML replies to the request with a html content.
func HTML(html string) any {
	return &content{
		name:    "index.html",
		content: bytes.NewReader([]byte(html)),
	}
}

// File replies to the request using the file content.
func File(name string) any {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			panic(&invalid{404, "file not found"})
		}
		panic(&invalid{500, err.Error()})
	}
	if fi.IsDir() {
		panic(&invalid{400, "is a directory"})
	}

	file, err := os.Open(name)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}

	return &content{path.Base(name), fi.ModTime(), file}
}

type fs struct {
	root     string
	fallback string
}

// FS replies to the request with the contents of the file system rooted at root.
func FS(root string, fallback string) any {
	fi, err := os.Lstat(root)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
	if !fi.IsDir() {
		panic(&invalid{500, "FS root is not a directory"})
	}
	return &fs{root, fallback}
}
