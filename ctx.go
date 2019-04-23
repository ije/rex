package rex

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/acl"
	"github.com/ije/rex/session"
)

type Context struct {
	W           http.ResponseWriter
	R           *http.Request
	URL         *URL
	State       *State
	session     session.Session
	privileges  map[string]struct{}
	basicUser   acl.BasicUser
	user        acl.User
	handles     []RESTHandle
	handleIndex int
	mux         *Mux
}

func (ctx *Context) Next() {
	ctx.handleIndex++
	if ctx.handleIndex >= len(ctx.handles) {
		return
	}

	if len(ctx.privileges) > 0 {
		var isGranted bool
		if ctx.user != nil {
			for _, pid := range ctx.user.Privileges() {
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

	// reset(prevent user chanage) the 'read-only' fields in context
	ctx.W = w
	ctx.R = r
	ctx.URL = url
	ctx.State = state
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

func (ctx *Context) Session() (sess session.Session) {
	cookieName := "x-session"
	if len(ctx.mux.SessionCookieName) > 0 {
		cookieName = ctx.mux.SessionCookieName
	}

	var sid string
	cookie, err := ctx.GetCookie(cookieName)
	if err == nil {
		sid = cookie.Value
	}

	sess = ctx.session
	if sess == nil {
		if ctx.mux.SessionManager == nil {
			panic(&ctxPanicError{"session manager is undefined"})
		}

		sess, err = ctx.mux.SessionManager.Get(sid)
		if err != nil {
			panic(&ctxPanicError{err.Error()})
		}

		if sess.SID() != sid {
			ctx.SetCookie(&http.Cookie{
				Name:     cookieName,
				Value:    sess.SID(),
				HttpOnly: true,
			})
		}
		ctx.session = sess
	}

	return
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

func (ctx *Context) FormArray(key string) (a []string) {
	if ctx.R.Form == nil {
		ctx.R.ParseMultipartForm(32 << 20) // 32m in memory
	}
	a, ok := ctx.R.Form[key]
	if !ok {
		a, _ = ctx.R.Form[key+"[]"]
	}
	return
}

func (ctx *Context) FormString(key string, defaultValue string) string {
	values := ctx.FormArray(key)
	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

func (ctx *Context) FormBool(key string, defaultValue bool) bool {
	values := ctx.FormArray(key)
	if len(values) > 0 {
		s := strings.ToLower(values[0])
		return s == "true" || s == "1"
	}
	return defaultValue
}

func (ctx *Context) FormFloat(key string, defaultValue float64) float64 {
	values := ctx.FormArray(key)
	if len(values) > 0 {
		f, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			return defaultValue
		}
		return f
	}
	return defaultValue
}

func (ctx *Context) FormInt(key string, defaultValue int64) int64 {
	values := ctx.FormArray(key)
	if len(values) > 0 {
		f, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return defaultValue
		}
		return f
	}
	return defaultValue
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

func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.W.Write(p)
}

func (ctx *Context) WriteString(s string) (n int, err error) {
	return ctx.W.Write([]byte(s))
}

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

func (ctx *Context) Ok(text string) {
	ctx.End(200, text)
}

func (ctx *Context) Error(err error) {
	if ctx.mux.Debug {
		ctx.End(500, err.Error())
	} else {
		if ctx.mux.Logger != nil {
			ctx.mux.Logger.Error(err)
		}
		ctx.End(500)
	}
}

func (ctx *Context) Html(html string) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.WriteString(html)
}

func (ctx *Context) Json(status int, v interface{}) {
	var data []byte
	var err error
	if ctx.mux.Debug {
		data, err = json.MarshalIndent(v, "", "\t")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(data) > 1024 && strings.Index(ctx.R.Header.Get("Accept-Encoding"), "gzip") > -1 {
		if w, ok := ctx.W.(*ResponseWriter); ok {
			gzw := newGzResponseWriter(w.rawWriter)
			defer gzw.Close()
			w.rawWriter = gzw
		}
	}

	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.W.WriteHeader(status)
	ctx.Write(data)
}

func (ctx *Context) File(filepath string) {
	if strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		for _, ext := range []string{"js", "js.map", "json", "css", "html", "htm", "xml", "svg", "txt"} {
			if strings.HasSuffix(strings.ToLower(filepath), "."+ext) {
				fi, err := os.Stat(filepath)
				if err != nil {
					if os.IsNotExist(err) {
						ctx.End(404)
					} else {
						ctx.Error(err)
					}
					return
				}
				if fi.Size() > 1024 {
					if w, ok := ctx.W.(*ResponseWriter); ok {
						gzw := newGzResponseWriter(w.rawWriter)
						defer gzw.Close()
						w.rawWriter = gzw
					}
				}
				break
			}
		}
	}
	http.ServeFile(ctx.W, ctx.R, filepath)
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
	ctx.SetHeader("Content-Type", "application/zip")
	if fi.IsDir() {
		utils.ZipFilesTo(path, ctx.W)
	} else {
		file, err := os.Open(path)
		if err != nil {
			ctx.Error(err)
			return
		}
		defer file.Close()

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			ctx.Error(err)
			return
		}

		archive := zip.NewWriter(ctx.W)
		defer archive.Close()

		gzwr, err := archive.CreateHeader(header)
		if err != nil {
			ctx.Error(err)
			return
		}

		io.Copy(gzwr, file)
	}
}

func (ctx *Context) BasicUser() acl.BasicUser {
	return ctx.basicUser
}

func (ctx *Context) User() acl.User {
	return ctx.user
}

func (ctx *Context) MustUser() acl.User {
	if ctx.user == nil {
		panic(&ctxPanicError{"the user is undefined"})
	}
	return ctx.user
}

func (ctx *Context) SetUser(user acl.User) {
	ctx.user = user
}
