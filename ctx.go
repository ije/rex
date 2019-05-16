package rex

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/acl"
)

type Context struct {
	W              http.ResponseWriter
	R              *http.Request
	URL            *URL
	State          *State
	handles        []RESTHandle
	handleIndex    int
	privileges     map[string]struct{}
	aclUser        acl.User
	basicUser      acl.BasicUser
	session        *ContextSession
	sessionManager *SessionManager
	rest           *REST
}

type Template interface {
	Execute(wr io.Writer, data interface{}) error
}

func (ctx *Context) Next() {
	ctx.handleIndex++
	if ctx.handleIndex >= len(ctx.handles) {
		return
	}

	if len(ctx.privileges) > 0 {
		var isGranted bool
		if ctx.aclUser != nil {
			for _, pid := range ctx.aclUser.Privileges() {
				_, isGranted = ctx.privileges[pid]
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

	// cache the 'read-only' fields in context firstly
	w, r, url, state := ctx.W, ctx.R, ctx.URL, ctx.State

	handle := ctx.handles[ctx.handleIndex]
	handle(ctx)

	// reset(to prevent user change) the 'read-only' fields in context
	ctx.W = w
	ctx.R = r
	ctx.URL = url
	ctx.State = state
}

func (ctx *Context) BasicUser() acl.BasicUser {
	return ctx.basicUser
}

func (ctx *Context) ACLUser() acl.User {
	return ctx.aclUser
}

func (ctx *Context) MustACLUser() acl.User {
	if ctx.aclUser == nil {
		panic(&contextPanicError{"ACL user of context is nil"})
	}
	return ctx.aclUser
}

func (ctx *Context) Session() *ContextSession {
	if ctx.sessionManager.Pool == nil {
		panic(&contextPanicError{"session pool is nil"})
	}

	if ctx.session == nil {
		sid := ctx.sessionManager.SIDStore.Get(ctx)
		sess, err := ctx.sessionManager.Pool.GetSession(sid)
		if err != nil {
			panic(&contextPanicError{err.Error()})
		}

		ctx.session = &ContextSession{sess}

		// restore sid
		if sess.SID() != sid {
			ctx.sessionManager.SIDStore.Set(ctx, sess.SID())
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
		cookie.Expires = time.Unix(0, 0)
		ctx.AddHeader("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) AddHeader(key string, value string) {
	ctx.W.Header().Add(key, value)
}

func (ctx *Context) SetHeader(key string, value string) {
	ctx.W.Header().Set(key, value)
}

func (ctx *Context) ParseMultipartForm(maxMemoryBytes int64) {
	if strings.Contains(ctx.R.Header.Get("Content-Type"), "/json") {
		form := url.Values{}
		var obj map[string]interface{}
		if json.NewDecoder(ctx.R.Body).Decode(&obj) == nil {
			for key, value := range obj {
				switch v := value.(type) {
				case []interface{}:
					for _, val := range v {
						form.Add(key, formatValue(val))
					}
				default:
					form.Set(key, formatValue(v))
				}
			}
		}
		ctx.R.Form = form
	} else {
		ctx.R.ParseMultipartForm(maxMemoryBytes)
	}
}

func formatValue(value interface{}) (str string) {
	switch v := value.(type) {
	case nil:
		str = "null"
	case bool:
		if v {
			str = "true"
		} else {
			str = "false"
		}
	case float64:
		str = fmt.Sprintf("%f", v)
	case string:
		str = v
	case map[string]interface{}:
		p, err := json.Marshal(v)
		if err == nil {
			str = string(p)
		}
	}
	return
}

func (ctx *Context) FormValues(key string) (a []string) {
	if ctx.R.Form == nil {
		ctx.R.ParseMultipartForm(32 << 20) // 32m in memory
	}
	a, ok := ctx.R.Form[key]
	if !ok {
		a, _ = ctx.R.Form[key+"[]"]
	}
	return
}

func (ctx *Context) FormValue(key string, defaultValue ...string) FormValue {
	values := ctx.FormValues(key)
	if len(values) > 0 {
		return FormValue(values[0])
	}
	if len(defaultValue) > 0 {
		return FormValue(defaultValue[0])
	}
	return FormValue("")
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

func (ctx *Context) Redirect(status int, url string) {
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
	if ctx.rest.SendError {
		ctx.End(500, err.Error())
	} else {
		ctx.End(500)
	}
	if ctx.rest.Logger != nil {
		ctx.rest.Logger.Println("[error]", err)
	}
}

func (ctx *Context) Html(html []byte) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.W.Write(html)
}

func (ctx *Context) RenderHTML(text string, data interface{}) {
	t, err := template.New("").Parse(text)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.RenderTemplate(t, data)
}

func (ctx *Context) RenderTemplate(template Template, data interface{}) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	template.Execute(ctx.W, data)
}

func (ctx *Context) Json(status int, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(data) > 1024 && strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		if w, ok := ctx.W.(*responseWriter); ok {
			gzw := newGzipWriter(w.rawWriter)
			defer gzw.Close()
			w.rawWriter = gzw
		}
	}

	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.W.WriteHeader(status)
	ctx.W.Write(data)
}

func (ctx *Context) File(filename string) {
	fi, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.End(404)
		} else {
			ctx.Error(err)
		}
		return
	}
	if fi.Size() > 1024 && strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		for _, ext := range []string{"html", "htm", "xml", "svg", "js", "jsx", "js.map", "ts", "tsx", "json", "css", "txt"} {
			if strings.HasSuffix(strings.ToLower(filename), "."+ext) {
				if w, ok := ctx.W.(*responseWriter); ok {
					gzw := newGzipWriter(w.rawWriter)
					defer gzw.Close()
					w.rawWriter = gzw
				}
				break
			}
		}
	}
	http.ServeFile(ctx.W, ctx.R, filename)
}

func (ctx *Context) Zip(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.End(404)
		} else {
			ctx.Error(err)
		}
		return
	}

	if fi.IsDir() {
		dir, err := filepath.Abs(path)
		if err != nil {
			ctx.Error(err)
			return
		}

		archive := zip.NewWriter(ctx.W)
		defer archive.Close()

		ctx.SetHeader("Content-Type", "application/zip")
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.TrimPrefix(path, dir), "/")
			if header.Name == "" {
				return nil
			}

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			gzw, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(gzw, file)
			return err
		})
	} else {
		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			ctx.Error(err)
			return
		}

		file, err := os.Open(path)
		if err != nil {
			ctx.Error(err)
			return
		}
		defer file.Close()

		archive := zip.NewWriter(ctx.W)
		defer archive.Close()

		gzw, err := archive.CreateHeader(header)
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.SetHeader("Content-Type", "application/zip")
		io.Copy(gzw, file)
	}
}
