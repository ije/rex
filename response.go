package rex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

type recoverError struct {
	status  int
	message string
}

// Error defines an error with status.
type Error struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func Err(status int, v ...string) *Error {
	var messsage string
	if len(v) > 0 {
		args := make([]interface{}, len(v))
		for i, s := range v {
			args[i] = s
		}
		messsage = fmt.Sprint(args...)
	} else if status > 100 && status < 600 {
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
func Redirect(url string, status int) interface{} {
	return &redirect{status, url}
}

type statusPlayload struct {
	status  int
	payload interface{}
}

// Status replies to the request using the payload in the status.
func Status(status int, payload interface{}) *statusPlayload {
	return &statusPlayload{status, payload}
}

// HTML replies to the request with a html content.
func HTML(html string) *contentful {
	return &contentful{
		name:    "index.html",
		mtime:   time.Now(),
		content: bytes.NewReader([]byte(html)),
	}
}

type Template interface {
	Name() string
	Execute(wr io.Writer, data interface{}) error
}

// Render renders the template with the given data.
func Render(t Template, data interface{}) interface{} {
	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, data); err != nil {
		panic(&recoverError{500, err.Error()})
	}

	return &contentful{
		name:    t.Name(),
		mtime:   time.Now(),
		content: bytes.NewReader(buf.Bytes()),
	}
}

type contentful struct {
	name    string
	mtime   time.Time
	content io.ReadSeeker
}

// Content replies to the request using the content in the provided ReadSeeker.
func Content(name string, mtime time.Time, r io.ReadSeeker) *contentful {
	return &contentful{name, mtime, r}
}

// File replies to the request using the file content.
func File(name string) *contentful {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			panic(&recoverError{404, "file not found"})
		}
		panic(&recoverError{500, err.Error()})
	}
	if fi.IsDir() {
		panic(&recoverError{400, "is a directory"})
	}

	file, err := os.Open(name)
	if err != nil {
		panic(&recoverError{500, err.Error()})
	}

	return &contentful{path.Base(name), fi.ModTime(), file}
}

type fs struct {
	root     string
	fallback string
}

// FS replies to the request with the contents of the file system rooted at root.
func FS(root string, fallback string) interface{} {
	fi, err := os.Lstat(root)
	if err != nil {
		panic(&recoverError{500, err.Error()})
	}
	if !fi.IsDir() {
		panic(&recoverError{500, "FS root is not a directory"})
	}
	return &fs{root, fallback}
}
