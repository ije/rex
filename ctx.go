package webx

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/webx/session"
	"github.com/julienschmidt/httprouter"
)

var globalSessionManager session.Manager = session.NewMemorySessionManager(time.Hour / 2)

func InitSessionManager(manager session.Manager) {
	if manager != nil {
		globalSessionManager = manager
	}
}

type Context struct {
	URL     *URL
	w       http.ResponseWriter
	r       *http.Request
	session session.Session
}

type URL struct {
	Params httprouter.Params
	*url.URL
}

func (ctx *Context) ResponseWriter() http.ResponseWriter {
	return ctx.w
}

func (ctx *Context) Request() *http.Request {
	return ctx.r
}

func (ctx *Context) Cookie(name string) (cookie *http.Cookie, err error) {
	return ctx.r.Cookie(name)
}

func (ctx *Context) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		ctx.w.Header().Add("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) RemoveCookie(cookie *http.Cookie) {
	if cookie != nil {
		cookie.Expires = time.Now().Add(-time.Second)
		ctx.w.Header().Add("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) Session() (sess session.Session, err error) {
	sessionCookieName := "x-session"
	if xs.App != nil && len(xs.App.sessionCookieName) > 0 {
		sessionCookieName = xs.App.sessionCookieName
	}

	var sid string
	if c, err := ctx.Cookie(sessionCookieName); err == nil {
		sid = c.Value
	}

	sess = ctx.session
	if sess == nil {
		sess, err = globalSessionManager.Get(sid)
		if err != nil {
			return
		}
		ctx.session = sess
	}

	if sid != sess.SID() {
		ctx.SetCookie(&http.Cookie{
			Name:     sessionCookieName,
			Value:    sess.SID(),
			HttpOnly: true,
		})
	}

	return
}

func (ctx *Context) ParseMultipartForm(maxMemoryBytes int64) {
	if strings.Contains(ctx.r.Header.Get("Content-Type"), "json") {
		var values map[string]interface{}
		if json.NewDecoder(ctx.r.Body).Decode(&values) == nil {
			form := url.Values{}
			for key, value := range values {
				switch v := value.(type) {
				case map[string]interface{}, []interface{}:
					b, err := json.Marshal(v)
					if err == nil {
						form.Set(key, string(b))
					}
				case string:
					form.Set(key, v)
				default:
					form.Set(key, fmt.Sprintf("%v", value))
				}
			}
			ctx.r.Form = form
			return
		}
	} else {
		ctx.r.ParseMultipartForm(maxMemoryBytes)
	}
}

func (ctx *Context) FormValues(key string) (values []string) {
	if ctx.r.Form == nil {
		ctx.ParseMultipartForm(32 << 20) // 32m in memory
	}
	values, ok := ctx.r.Form[key]
	if !ok {
		values, _ = ctx.r.Form[key+"[]"]
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
		b = s != "false" && s != "0" && s != "no" && s != "disable"
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

func (ctx *Context) FormJSON(key string) (value map[string]interface{}, err error) {
	if s := strings.TrimSpace(ctx.FormString(key)); len(s) > 0 {
		err = json.Unmarshal([]byte(s), &value)
	}
	return
}

func (ctx *Context) RemoteIp() (ip string) {
	ip = ctx.r.Header.Get("X-Real-IP")
	if len(ip) == 0 {
		ip = ctx.r.Header.Get("X-Forwarded-For")
		if len(ip) > 0 {
			ip, _ = utils.SplitByFirstByte(ip, ',')
			ip = strings.TrimSpace(ip)
		} else {
			ip = ctx.r.RemoteAddr
		}
	}

	ip, _ = utils.SplitByLastByte(ip, ':')
	return
}

func (ctx *Context) Authenticate(realm string, authHandle func(user string, password string) (ok bool, err error)) (ok bool, err error) {
	if authField := ctx.r.Header.Get("Authorization"); len(authField) > 0 {
		if authType, combination := utils.SplitByFirstByte(authField, ' '); len(combination) > 0 {
			switch authType {
			case "Basic":
				authInfo, e := base64.StdEncoding.DecodeString(combination)
				if e != nil {
					return
				}

				user, password := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err = authHandle(user, password)
				return
			}
		}
	}

	ctx.w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
	ctx.w.WriteHeader(401)
	return
}

func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.w.Write(p)
}

func (ctx *Context) WriteJSON(status int, data interface{}) (n int, err error) {
	var jsonData []byte
	if config.Debug {
		jsonData, err = json.MarshalIndent(data, "", "\t")
	} else {
		jsonData, err = json.Marshal(data)
	}
	if err != nil {
		ctx.Error(err)
		return
	}

	var wr io.Writer = ctx.w
	wh := ctx.w.Header()
	if len(jsonData) > 1024 && strings.Index(ctx.r.Header.Get("Accept-Encoding"), "gzip") > -1 {
		wh.Set("Content-Encoding", "gzip")
		wh.Set("Vary", "Accept-Encoding")
		gz, _ := gzip.NewWriterLevel(ctx.w, gzip.BestSpeed)
		defer gz.Close()
		wr = gz
	}
	wh.Set("Content-Type", "application/json; charset=utf-8")
	ctx.w.WriteHeader(status)
	return wr.Write(jsonData)
}

func (ctx *Context) End(status int, a ...string) {
	wh := ctx.w.Header()
	if _, ok := wh["Content-Type"]; !ok {
		wh.Set("Content-Type", "text/plain; charset=utf-8")
	}
	ctx.w.WriteHeader(status)
	var text string
	if len(a) > 0 {
		text = strings.Join(a, " ")
	} else {
		text = http.StatusText(status)
	}
	ctx.w.Write([]byte(text))
}

func (ctx *Context) Error(err error) {
	if config.Debug {
		ctx.End(500, err.Error())
	} else {
		ctx.End(500)
	}
}
