package rex

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
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
	handles     []Handle
	handleIndex int
	permissions map[string]struct{}
	aclUser     ACLUser
	basicUser   BasicUser
	valueStore  sync.Map
	sidStore    session.SIDStore
	sessionPool session.Pool
	session     *ContextSession
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
	w, r, url := ctx.W, ctx.R, ctx.URL

	handle(ctx)

	// restore(to prevent user change) the 'read-only' fields in context
	ctx.W = w
	ctx.R = r
	ctx.URL = url
}

func (ctx *Context) Value(key string) (value interface{}, ok bool) {
	return ctx.valueStore.Load(key)
}

func (ctx *Context) StoreValue(key string, value interface{}) {
	ctx.valueStore.Store(key, value)
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

func (ctx *Context) Session() *ContextSession {
	if ctx.sessionPool == nil {
		panic(&contextPanicError{"session pool is nil"})
	}

	if ctx.session == nil {
		sid := ctx.sidStore.Get(ctx.R)
		sess, err := ctx.sessionPool.GetSession(sid)
		if err != nil {
			panic(&contextPanicError{err.Error()})
		}

		ctx.session = &ContextSession{sess}

		// restore sid
		if sess.SID() != sid {
			ctx.sidStore.Put(ctx.W, sess.SID())
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

func (ctx *Context) GetHeader(key string) string {
	return ctx.R.Header.Get(key)
}

func (ctx *Context) AddHeader(key string, value string) {
	ctx.W.Header().Add(key, value)
}

func (ctx *Context) SetHeader(key string, value string) {
	ctx.W.Header().Set(key, value)
}

// FormValue returns the first value for the named component of the POST,
// PATCH, or PUT request body, or returns the first value for the named component of the request url query
func (ctx *Context) FormValue(key string) string {
	switch ctx.R.Method {
	case "POST", "PUT", "PATCH":
		return ctx.R.PostFormValue(key)
	default:
		return ctx.R.FormValue(key)
	}
}

func (ctx *Context) FormIntValue(key string) (int64, error) {
	v := strings.TrimSpace(ctx.FormValue(key))
	if v == "" {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func (ctx *Context) FormFloatValue(key string) (float64, error) {
	v := strings.TrimSpace(ctx.FormValue(key))
	if v == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(v, 64)
}

func (ctx *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.R.FormFile(key)
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

func (ctx *Context) Ok(text string) {
	ctx.End(200, text)
}

func (ctx *Context) Error(err error) {
	if ctx.rest.debug {
		ctx.End(500, err.Error())
	} else {
		ctx.End(500)
	}
	if ctx.rest.Logger != nil {
		ctx.rest.Logger.Println("[error]", err)
	}
}

func (ctx *Context) json(status int, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(data) > 1024 {
		ctx.EnableGzip()
	}

	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.W.WriteHeader(status)
	ctx.W.Write(data)
}

func (ctx *Context) JSON(v interface{}) {
	ctx.json(200, v)
}

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
		if ctx.rest.debug {
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

func (ctx *Context) HTML(html string) {
	if len(html) > 1024 {
		ctx.EnableGzip()
	}
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.W.Write([]byte(html))
}

func (ctx *Context) RenderHTML(html string, data interface{}) {
	t, err := template.New("").Parse(html)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.Render(t, data)
}

func (ctx *Context) Render(template Template, data interface{}) {
	buf := bytes.NewBuffer(nil)
	err := template.Execute(buf, data)
	if err != nil {
		ctx.Error(err)
		return
	}

	if buf.Len() > 1024 {
		ctx.EnableGzip()
	}
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	io.Copy(ctx.W, buf)
}

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
	if fi.Size() > 1024 {
		switch strings.TrimLeft(path.Ext(name), ".") {
		case "html", "htm", "xml", "svg", "js", "json", "css", "txt", "map":
			ctx.EnableGzip()
		}
	}
	http.ServeFile(ctx.W, ctx.R, name)
}

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
			ctx.EnableGzip()
		}
	}
	http.ServeContent(ctx.W, ctx.R, name, modtime, content)
}

func (ctx *Context) EnableGzip() {
	if strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		if w, ok := ctx.W.(*responseWriter); ok {
			if _, ok = w.rawWriter.(*gzipResponseWriter); !ok {
				w.rawWriter = newGzipWriter(w.rawWriter)
			}
		}
	}
}

type ContextSession struct {
	sess session.Session
}

func (s *ContextSession) SID() string {
	return s.sess.SID()
}

func (s *ContextSession) Has(key string) bool {
	ok, err := s.sess.Has(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return ok
}

func (s *ContextSession) Get(key string) interface{} {
	value, err := s.sess.Get(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return value
}

func (s *ContextSession) Set(key string, value interface{}) {
	err := s.sess.Set(key, value)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

func (s *ContextSession) Delete(key string) {
	err := s.sess.Delete(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

func (s *ContextSession) Flush() {
	err := s.sess.Flush()
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}
