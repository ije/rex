package rex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"text/template"
	"time"
)

// Response defines the response interface.
type Response any

type invalid struct {
	status  int
	message string
}

// Error defines an error with status.
type Error struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Err returns an error with status.
func Err(status int, v ...string) Response {
	if status < 400 || status >= 600 {
		panic("invalid status code")
	}
	var messsage string
	if len(v) > 0 {
		args := make([]any, len(v))
		for i, s := range v {
			args[i] = s
		}
		messsage = fmt.Sprint(args...)
	} else {
		messsage = http.StatusText(status)
	}
	return &Error{
		Status:  status,
		Message: messsage,
	}
}

type redirect struct {
	status int
	url    string
}

// Redirect replies to the request with a redirect to url,
// which may be a path relative to the request path.
func Redirect(url string, status int) Response {
	if url == "" {
		url = "/"
	}
	if status < 300 || status >= 400 {
		status = 302
	}
	return &redirect{status, url}
}

type statusd struct {
	code    int
	content any
}

// Status replies to the request using the payload in the status.
func Status(status int, content any) Response {
	return &statusd{status, content}
}

// HTML replies to the request with a html content.
func HTML(html string) Response {
	return &content{
		name:    "index.html",
		content: bytes.NewReader([]byte(html)),
	}
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
func Render(t Template, data any) Response {
	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, data); err != nil {
		panic(&invalid{500, err.Error()})
	}

	return &content{
		name:    t.Name(),
		content: bytes.NewReader(buf.Bytes()),
	}
}

type content struct {
	name    string
	mtime   time.Time
	content io.ReadSeeker
}

// Content replies to the request using the content in the provided ReadSeeker.
func Content(name string, mtime time.Time, r io.ReadSeeker) Response {
	return &content{name, mtime, r}
}

// File replies to the request using the file content.
func File(name string) Response {
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
func FS(root string, fallback string) Response {
	fi, err := os.Lstat(root)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
	if !fi.IsDir() {
		panic(&invalid{500, "FS root is not a directory"})
	}
	return &fs{root, fallback}
}
