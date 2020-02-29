package rex

import (
	"bytes"
	"encoding/json"
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

type Context struct {
	W           http.ResponseWriter
	R           *http.Request
	URL         *URL
	Form        *Form
	handles     []Handle
	handleIndex int
	values      sync.Map
	permissions map[string]struct{}
	aclUser     ACLUser
	basicUser   BasicUser
	sidManager  session.SIDManager
	sessionPool session.Pool
	session     *Session
	rest        *REST
}

func (ctx *Context) Next() {
	ctx.handleIndex++
	if ctx.handleIndex >= len(ctx.handles) {
		return
	}

	if len(ctx.permissions) > 0 {
		var isGranted bool
		if ctx.aclUser != nil {
			for _, id := range ctx.aclUser.Permissions() {
				_, isGranted = ctx.permissions[id]
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

	// cache the 'read-only' fields in context firstly
	w, r, url, form := ctx.W, ctx.R, ctx.URL, ctx.Form

	handle(ctx)

	// restore(to prevent user change) the 'read-only' fields in context
	ctx.W = w
	ctx.R = r
	ctx.URL = url
	ctx.Form = form
}

func (ctx *Context) BasicUser() BasicUser {
	return ctx.basicUser
}

func (ctx *Context) ACLUser() ACLUser {
	return ctx.aclUser
}

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

func (ctx *Context) Session() *Session {
	if ctx.sessionPool == nil {
		panic(&contextPanicError{500, "session pool is nil"})
	}

	if ctx.session == nil {
		sid := ctx.sidManager.Get(ctx.R)
		sess, err := ctx.sessionPool.GetSession(sid)
		if err != nil {
			panic(&contextPanicError{500, err.Error()})
		}

		ctx.session = &Session{sess}

		// restore sid
		if sess.SID() != sid {
			ctx.sidManager.Put(ctx.W, sess.SID())
		}
	}

	return ctx.session
}

func (ctx *Context) GetCookie(name string) (cookie *http.Cookie, err error) {
	return ctx.R.Cookie(name)
}

func (ctx *Context) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		ctx.AddHeader("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) RemoveCookie(cookie *http.Cookie) {
	if cookie != nil {
		cookie.Value = "-"
		cookie.Expires = time.Unix(0, 0)
		ctx.SetCookie(cookie)
	}
}

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

func (ctx *Context) RemoteIP() string {
	ip := ctx.R.Header.Get("X-Real-IP")
	if len(ip) == 0 {
		ip = ctx.R.Header.Get("X-Forwarded-For")
		if len(ip) > 0 {
			ip, _ = utils.SplitByFirstByte(ip, ',')
		} else {
			ip = ctx.R.RemoteAddr
		}
	}
	ip, _ = utils.SplitByLastByte(ip, ':')
	return strings.TrimSpace(ip)
}

func (ctx *Context) Redirect(url string, status int) {
	http.Redirect(ctx.W, ctx.R, url, status)
}

func (ctx *Context) IfModified(modtime time.Time, then func()) {
	if t, err := time.Parse(http.TimeFormat, ctx.R.Header.Get("If-Modified-Since")); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.End(http.StatusNotModified)
		return
	}

	ctx.SetHeader("Last-Modified", modtime.Format(http.TimeFormat))
	then()
}

func (ctx *Context) IfNotMatch(etag string, then func()) {
	if ctx.R.Header.Get("If-Not-Match") == etag {
		ctx.End(http.StatusNotModified)
		return
	}

	ctx.SetHeader("ETag", etag)
	then()
}

func (ctx *Context) End(status int, a ...string) {
	wh := ctx.W.Header()
	if _, ok := wh["Content-Type"]; !ok {
		wh.Set("Content-Type", "text/plain; charset=utf-8")
	}
	ctx.W.WriteHeader(status)
	if len(a) > 0 {
		ctx.W.Write([]byte(strings.Join(a, " ")))
	} else {
		ctx.W.Write([]byte(http.StatusText(status)))
	}
}

// Ok replies to the request the plain text with 200 status.
func (ctx *Context) Ok(text string) {
	ctx.End(200, text)
}

// JSON replies to the request as a json.
func (ctx *Context) JSON(v interface{}) {
	ctx.json(200, v)
}

// JSONError replies to the request a json error.
func (ctx *Context) JSONError(err error) {
	inv, ok := err.(*InvalidError)
	if ok {
		ctx.json(inv.Code, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    inv.Code,
				"message": inv.Message,
			},
		})
	} else {
		message := "internal server error"
		if ctx.rest.sendError {
			message = err.Error()
		}
		ctx.json(500, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    500,
				"message": message,
			},
		})
		if ctx.rest.Logger != nil {
			ctx.rest.Logger.Println("[error]", err)
		}
	}
}

func (ctx *Context) json(status int, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(data) > 1024 {
		ctx.enableGzip()
	}

	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.W.WriteHeader(status)
	ctx.W.Write(data)
}

// Error replies to the request a internal server error.
// if debug is enable, replies the error message.
func (ctx *Context) Error(err error) {
	if ctx.rest.sendError {
		ctx.End(500, err.Error())
	} else {
		ctx.End(500)
	}
	if ctx.rest.Logger != nil {
		ctx.rest.Logger.Println("[error]", err)
	}
}

// HTML replies to the request as a html.
func (ctx *Context) HTML(html string) {
	if len(html) > 1024 {
		ctx.enableGzip()
	}
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.W.Write([]byte(html))
}

// RenderHTML applies a unparsed html template with the specified data object,
// replies to the request.
func (ctx *Context) RenderHTML(html string, data interface{}) {
	t, err := template.New("").Parse(html)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.Render(t, data)
}

// Render applies a parsed template with the specified data object,
// replies to the request.
func (ctx *Context) Render(template Template, data interface{}) {
	buf := bytes.NewBuffer(nil)
	err := template.Execute(buf, data)
	if err != nil {
		ctx.Error(err)
		return
	}

	if buf.Len() > 1024 {
		ctx.enableGzip()
	}
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	io.Copy(ctx.W, buf)
}

// File replies to the request with the contents of the named
// file or directory.
func (ctx *Context) File(name string) {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.End(404)
		} else {
			ctx.Error(err)
		}
		return
	}
	if !fi.IsDir() && fi.Size() > 1024 {
		switch strings.TrimLeft(path.Ext(name), ".") {
		case "html", "htm", "xml", "svg", "js", "json", "css", "txt", "map":
			ctx.enableGzip()
		}
	}
	http.ServeFile(ctx.W, ctx.R, name)
}

// Content replies to the request using the content in the
// provided ReadSeeker. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
func (ctx *Context) Content(name string, modtime time.Time, content io.ReadSeeker) {
	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		ctx.Error(err)
		return
	}
	_, err = content.Seek(0, io.SeekStart)
	if err != nil {
		ctx.Error(err)
		return
	}
	if size > 1024 {
		switch strings.TrimLeft(path.Ext(name), ".") {
		case "html", "htm", "xml", "svg", "js", "json", "css", "txt", "map":
			ctx.enableGzip()
		}
	}
	http.ServeContent(ctx.W, ctx.R, name, modtime, content)
}

func (ctx *Context) enableGzip() {
	if strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		if w, ok := ctx.W.(*responseWriter); ok {
			if _, ok = w.rawWriter.(*gzipResponseWriter); !ok {
				w.rawWriter = newGzipWriter(w.rawWriter)
			}
		}
	}
}
