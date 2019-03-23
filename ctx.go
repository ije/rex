package rex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/acl"
	"github.com/ije/rex/session"
	"github.com/julienschmidt/httprouter"
)

type URL struct {
	Params httprouter.Params
	*url.URL
}

type Context struct {
	W       http.ResponseWriter
	R       *http.Request
	URL     *URL
	State   *State
	session session.Session
	user    acl.User
	mux     *Mux
}

func (ctx *Context) GetCookie(name string) (cookie *http.Cookie, err error) {
	return ctx.R.Cookie(name)
}

func (ctx *Context) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		ctx.W.Header().Add("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) RemoveCookie(cookie *http.Cookie) {
	if cookie != nil {
		cookie.Expires = time.Now().Add(-(1000 * time.Hour))
		ctx.W.Header().Add("Set-Cookie", cookie.String())
	}
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
			panic(&initSessionError{"missing session manager"})
		}

		sess, err = ctx.mux.SessionManager.Get(sid)
		if err != nil {
			panic(&initSessionError{err.Error()})
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

func (ctx *Context) FormValues(key string) (values []string) {
	if ctx.R.Form == nil {
		ctx.R.ParseMultipartForm(32 << 20) // 32m in memory
	}
	values, ok := ctx.R.Form[key]
	if !ok {
		values, _ = ctx.R.Form[key+"[]"]
	}
	return
}

func (ctx *Context) FormString(key string) (value string) {
	values := ctx.FormValues(key)
	if len(values) > 0 {
		value = values[0]
	}
	return
}

func (ctx *Context) FormBool(key string) (b bool) {
	s := strings.TrimSpace(ctx.FormString(key))
	if len(s) > 0 {
		s = strings.ToLower(s)
		b = s == "true" || s == "1"
	}
	return
}

func (ctx *Context) FormNumber(key string) (n float64, err error) {
	s := strings.TrimSpace(ctx.FormString(key))
	if len(s) == 0 {
		err = strconv.ErrSyntax
		return
	}

	n, err = strconv.ParseFloat(s, 64)
	return
}

func (ctx *Context) FormInt(key string) (i int, err error) {
	n, err := ctx.FormNumber(key)
	if err != nil {
		return
	}

	i = int(n)
	return
}

func (ctx *Context) RemoteIP() (ip string) {
	ip = ctx.R.Header.Get("X-Real-IP")
	if len(ip) == 0 {
		ip = ctx.R.Header.Get("X-Forwarded-For")
		if len(ip) > 0 {
			ip, _ = utils.SplitByFirstByte(ip, ',')
		} else {
			ip = ctx.R.RemoteAddr
		}
	}
	ip, _ = utils.SplitByLastByte(strings.TrimSpace(ip), ':')
	return
}

func (ctx *Context) Redirect(url string, code int) {
	http.Redirect(ctx.W, ctx.R, url, code)
}

func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.W.Write(p)
}

func (ctx *Context) WriteString(s string) (n int, err error) {
	return ctx.W.Write([]byte(s))
}

func (ctx *Context) WriteJSON(data interface{}) (n int, err error) {
	return ctx.writeJSON(200, data)
}

func (ctx *Context) WriteStatusJSON(status int, data interface{}) (n int, err error) {
	return ctx.writeJSON(status, data)
}

func (ctx *Context) writeJSON(status int, data interface{}) (n int, err error) {
	var jsonData []byte
	if ctx.mux.Debug {
		jsonData, err = json.MarshalIndent(data, "", "\t")
	} else {
		jsonData, err = json.Marshal(data)
	}
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(jsonData) > 1024 && strings.Index(ctx.R.Header.Get("Accept-Encoding"), "gzip") > -1 {
		if w, ok := ctx.W.(*ResponseWriter); ok {
			gzw := newGzResponseWriter(w.rawWriter)
			defer gzw.Close()
			w.rawWriter = gzw
		}
	}

	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.WriteHeader(status)
	return ctx.Write(jsonData)
}

func (ctx *Context) IfModified(modtime time.Time, then func()) {
	if t, err := time.Parse(http.TimeFormat, ctx.R.Header.Get("If-Modified-Since")); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.End(http.StatusNotModified)
		return
	}

	ctx.W.Header().Set("Last-Modified", modtime.Format(http.TimeFormat))
	then()
}

func (ctx *Context) ServeFile(name string) {
	if strings.Contains(ctx.R.Header.Get("Accept-Encoding"), "gzip") {
		switch strings.ToLower(strings.TrimPrefix(path.Ext(name), ".")) {
		case "js", "css", "html", "htm", "xml", "svg", "json", "txt":
			fi, err := os.Stat(name)
			if err != nil {
				if os.IsNotExist(err) {
					ctx.End(404)
				} else {
					ctx.End(500)
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
		}
	}
	http.ServeFile(ctx.W, ctx.R, name)
}

func (ctx *Context) End(status int, a ...interface{}) {
	wh := ctx.W.Header()
	if _, ok := wh["Content-Type"]; !ok {
		wh.Set("Content-Type", "text/plain; charset=utf-8")
	}
	ctx.W.WriteHeader(status)
	if len(a) > 0 {
		ctx.WriteString(fmt.Sprint(a...))
	} else {
		ctx.WriteString(http.StatusText(status))
	}
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

func (ctx *Context) Authenticate(realm string, handle func(user string, password string) (ok bool, err error)) (ok bool, err error) {
	if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
		if authType, combination := utils.SplitByFirstByte(auth, ' '); len(combination) > 0 {
			switch authType {
			case "Basic":
				authInfo, e := base64.StdEncoding.DecodeString(combination)
				if e != nil {
					return
				}

				user, password := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err = handle(user, password)
				return
			}
		}
	}

	ctx.W.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
	ctx.W.WriteHeader(401)
	return
}

func (ctx *Context) User() acl.User {
	return ctx.user
}

func (ctx *Context) SetUser(user acl.User) {
	ctx.user = user
}
