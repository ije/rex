package rex

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
)

// A Context to handle http requests.
type Context struct {
	W            http.ResponseWriter
	R            *http.Request
	URL          *URL
	Form         *Form
	handles      []Handle
	handleIndex  int
	values       sync.Map
	acl          map[string]struct{}
	aclUser      ACLUser
	sidStore     session.SIDStore
	sessionPool  session.Pool
	session      *Session
	sendError    bool
	errorType    string
	logger       Logger
	accessLogger Logger
}

// A ACLUser contains the Permissions method that returns the acl permissions
type ACLUser interface {
	Permissions() []string
}

// Next calls the next handle.
func (ctx *Context) Next() {
	ctx.handleIndex++
	if ctx.handleIndex >= len(ctx.handles) {
		return
	}

	if len(ctx.acl) > 0 {
		var isGranted bool
		if ctx.aclUser != nil {
			for _, id := range ctx.aclUser.Permissions() {
				_, isGranted = ctx.acl[id]
				if isGranted {
					break
				}
			}
		}
		if !isGranted {
			ctx.End(http.StatusUnauthorized)
			return
		}
	}

	handle := ctx.handles[ctx.handleIndex]

	// cache the public fields in context
	w, r, url, form := ctx.W, ctx.R, ctx.URL, ctx.Form

	handle(ctx)

	// restore(to prevent user change) the public fields in context
	ctx.W = w
	ctx.R = r
	ctx.URL = url
	ctx.Form = form
}

// BasicUser returns the basic user
func (ctx *Context) BasicUser() interface{} {
	v, _ := ctx.values.Load("__BASIC_USER__")
	return v
}

// ACLUser returns the acl user
func (ctx *Context) ACLUser() ACLUser {
	return ctx.aclUser
}

// SetACLUser sets the acl user
func (ctx *Context) SetACLUser(user ACLUser) {
	ctx.aclUser = user
}

// GetValue returns the value stored in the values for a key, or nil if no
// value is present.
func (ctx *Context) GetValue(key string) (interface{}, bool) {
	return ctx.values.Load(key)
}

// StoreValue sets the value for a key.
func (ctx *Context) StoreValue(key string, value interface{}) {
	ctx.values.Store(key, value)
}

// Session returns the session if it is undefined then create a new one.
func (ctx *Context) Session() *Session {
	if ctx.sessionPool == nil {
		panic(&contextPanicError{"session pool is nil", 500})
	}

	if ctx.session == nil {
		sid := ctx.sidStore.Get(ctx.R)
		sess, err := ctx.sessionPool.GetSession(sid)
		if err != nil {
			panic(&contextPanicError{err.Error(), 500})
		}

		ctx.session = &Session{sess}

		if sess.SID() != sid {
			ctx.sidStore.Put(ctx.W, sess.SID())
		}
	}

	return ctx.session
}

// GetCookie returns the cookie by name.
func (ctx *Context) GetCookie(name string) (cookie *http.Cookie, err error) {
	return ctx.R.Cookie(name)
}

// SetCookie sets a cookie.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		ctx.AddHeader("Set-Cookie", cookie.String())
	}
}

// RemoveCookie removes the cookie.
func (ctx *Context) RemoveCookie(cookie *http.Cookie) {
	if cookie != nil {
		cookie.Value = "-"
		cookie.Expires = time.Unix(0, 0)
		ctx.SetCookie(cookie)
	}
}

// RemoveCookieByName removes the cookie by name.
func (ctx *Context) RemoveCookieByName(name string) {
	ctx.SetCookie(&http.Cookie{
		Name:    name,
		Value:   "-",
		Expires: time.Unix(0, 0),
	})
}

// AddHeader adds the key, value pair to the header of response writer.
func (ctx *Context) AddHeader(key string, value string) {
	ctx.W.Header().Add(key, value)
}

// SetHeader sets the header of response writer entries associated with key to the
// single element value.
func (ctx *Context) SetHeader(key string, value string) {
	ctx.W.Header().Set(key, value)
}

// RemoteIP returns the remote client IP
func (ctx *Context) RemoteIP() string {
	ip := ctx.R.Header.Get("X-Real-IP")
	if ip == "" {
		ip = ctx.R.Header.Get("X-Forwarded-For")
		if ip != "" {
			ip, _ = utils.SplitByFirstByte(ip, ',')
		} else {
			ip = ctx.R.RemoteAddr
		}
	}
	ip, _ = utils.SplitByLastByte(ip, ':')
	return ip
}

// Redirect replies to the request with a redirect to url,
// which may be a path relative to the request path.
func (ctx *Context) Redirect(url string, status int) {
	http.Redirect(ctx.W, ctx.R, url, status)
}

// IfModified handles caches by modified date.
func (ctx *Context) IfModified(modtime time.Time, then func()) {
	if t, err := time.Parse(http.TimeFormat, ctx.R.Header.Get("If-Modified-Since")); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.End(http.StatusNotModified)
		return
	}

	ctx.SetHeader("Last-Modified", modtime.Format(http.TimeFormat))
	then()
}

// IfNotMatch handles caches by etag.
func (ctx *Context) IfNotMatch(etag string, then func()) {
	if ctx.R.Header.Get("If-Not-Match") == etag {
		ctx.End(http.StatusNotModified)
		return
	}

	ctx.SetHeader("ETag", etag)
	then()
}

// JSON replies to the request as a json.
func (ctx *Context) JSON(v interface{}) {
	ctx.json(v, 200)
}

// json replies to the request as a json with status.
func (ctx *Context) json(v interface{}, status int) {
	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.enableGzip(".json")
	ctx.W.WriteHeader(status)
	err := json.NewEncoder(ctx.W).Encode(v)
	if err != nil {
		fmt.Println(err)
		ctx.Error(err.Error(), 500)
	}
}

// End replies to the request the status.
func (ctx *Context) End(status int, a ...string) {
	wh := ctx.W.Header()
	if _, ok := wh["Content-Type"]; !ok {
		wh.Set("Content-Type", "text/plain; charset=utf-8")
	}
	ctx.W.WriteHeader(status)
	if len(a) > 0 {
		ctx.Write([]byte(strings.Join(a, " ")))
	} else {
		ctx.Write([]byte(http.StatusText(status)))
	}
}

// Ok replies to the request the plain text with 200 status.
func (ctx *Context) Ok(text string) {
	ctx.End(200, text)
}

// Error replies to the request a internal server error.
// if debug is enable, replies the error message.
func (ctx *Context) Error(message string, status int) {
	if status >= 500 && !ctx.sendError {
		message = http.StatusText(status)
	}
	if ctx.errorType == "json" {
		ctx.json(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    status,
				"message": message,
			},
		}, status)
	} else {
		ctx.End(status, message)
	}
	if ctx.logger != nil && status >= 500 {
		ctx.logger.Print("[error]", message)
	}
}

// HTML replies to the request as a html.
func (ctx *Context) HTML(html string) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.enableGzip(".html")
	ctx.Write([]byte(html))
}

// RenderHTML applies a unparsed html template with the specified data object,
// replies to the request.
func (ctx *Context) RenderHTML(html string, data interface{}) {
	t, err := template.New("").Parse(html)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}
	ctx.Render(t, data)
}

// Render applies a parsed template with the specified data object,
// replies to the request.
func (ctx *Context) Render(template Template, data interface{}) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.enableGzip(".html")
	err := template.Execute(ctx.W, data)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}
}

// Content replies to the request using the content in the
// provided ReadSeeker. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
func (ctx *Context) Content(name string, modtime time.Time, content io.ReadSeeker) {
	ctx.enableGzip(path.Ext(name))
	http.ServeContent(ctx.W, ctx.R, name, modtime, content)
}

// File replies to the request with the contents of the named
// file or directory.
func (ctx *Context) File(name string) {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.End(404)
		} else {
			ctx.Error(err.Error(), 500)
		}
		return
	}
	if !fi.IsDir() {
		ctx.File(path.Join(name, "index.html"))
		return
	}
	ctx.enableGzip(path.Ext(name))
	http.ServeFile(ctx.W, ctx.R, name)
}

// Write implements the io.Writer.
func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.W.Write(p)
}

// enableGzip switches the gzip writer
func (ctx *Context) enableGzip(ext string) {
	if strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
		case "html", "htm", "xml", "svg", "css", "json", "js", "jsx", "mjs", "ts", "tsx", "map", "md", "txt":
			if w, ok := ctx.W.(*responseWriter); ok {
				if _, ok = w.rawWriter.(*gzipResponseWriter); !ok {
					w.rawWriter = newGzipWriter(w.rawWriter)
				}
			}
		}
	}
}
