package rex

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
	"github.com/ije/rex/acl"
	"github.com/ije/rex/session"
	"github.com/julienschmidt/httprouter"
)

type URL struct {
	Params httprouter.Params
	*url.URL
}

type Context struct {
	App            *App
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	URL            *URL
	State          *State
	session        session.Session
	user           acl.User
	mux            *Mux
}

func (ctx *Context) AddHeader(key string, value string) {
	ctx.ResponseWriter.Header().Add(key, value)
}

func (ctx *Context) SetHeader(key string, value string) {
	ctx.ResponseWriter.Header().Set(key, value)
}

func (ctx *Context) DelHeader(key string) {
	ctx.ResponseWriter.Header().Del(key)
}

func (ctx *Context) WriteHeader(status int) {
	ctx.ResponseWriter.WriteHeader(status)
}

func (ctx *Context) Cookie(name string) (cookie *http.Cookie, err error) {
	return ctx.Request.Cookie(name)
}

func (ctx *Context) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		ctx.AddHeader("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) RemoveCookie(cookie *http.Cookie) {
	if cookie != nil {
		cookie.Expires = time.Now().Add(-(365 * 24 * time.Hour))
		ctx.AddHeader("Set-Cookie", cookie.String())
	}
}

func (ctx *Context) Session() (sess session.Session) {
	cookieName := "x-session"
	if len(ctx.mux.SessionCookieName) > 0 {
		cookieName = ctx.mux.SessionCookieName
	}

	var sid string
	cookie, err := ctx.Cookie(cookieName)
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
	if strings.Contains(ctx.Request.Header.Get("Content-Type"), "json") {
		var values map[string]interface{}
		if json.NewDecoder(ctx.Request.Body).Decode(&values) == nil {
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
			ctx.Request.Form = form
			return
		}
	} else {
		ctx.Request.ParseMultipartForm(maxMemoryBytes)
	}
}

func (ctx *Context) FormValues(key string) (values []string) {
	if ctx.Request.Form == nil {
		ctx.ParseMultipartForm(32 << 20) // 32m in memory
	}
	values, ok := ctx.Request.Form[key]
	if !ok {
		values, _ = ctx.Request.Form[key+"[]"]
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

func (ctx *Context) RemoteIP() (ip string) {
	ip = ctx.Request.Header.Get("X-Real-IP")
	if len(ip) == 0 {
		ip = ctx.Request.Header.Get("X-Forwarded-For")
		if len(ip) > 0 {
			ip, _ = utils.SplitByFirstByte(ip, ',')
			ip = strings.TrimSpace(ip)
		} else {
			ip = ctx.Request.RemoteAddr
		}
	}

	ip, _ = utils.SplitByLastByte(ip, ':')
	return
}

func (ctx *Context) Authenticate(realm string, authHandle func(user string, password string) (ok bool, err error)) (ok bool, err error) {
	if authField := ctx.Request.Header.Get("Authorization"); len(authField) > 0 {
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

	ctx.SetHeader("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
	ctx.WriteHeader(401)
	return
}

func (ctx *Context) Redirect(url string, code int) {
	http.Redirect(ctx.ResponseWriter, ctx.Request, url, code)
}

func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.ResponseWriter.Write(p)
}

func (ctx *Context) WriteString(s string) (n int, err error) {
	return ctx.ResponseWriter.Write([]byte(s))
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

	var wr io.Writer = ctx.ResponseWriter
	wh := ctx.ResponseWriter.Header()
	if len(jsonData) > 1024 && strings.Index(ctx.Request.Header.Get("Accept-Encoding"), "gzip") > -1 {
		wh.Set("Content-Encoding", "gzip")
		wh.Set("Vary", "Accept-Encoding")
		gz, _ := gzip.NewWriterLevel(ctx.ResponseWriter, gzip.BestSpeed)
		defer gz.Close()
		wr = gz
	}
	wh.Set("Content-Type", "application/json; charset=utf-8")
	ctx.ResponseWriter.WriteHeader(status)
	return wr.Write(jsonData)
}

func (ctx *Context) End(status int, a ...string) {
	wh := ctx.ResponseWriter.Header()
	if _, ok := wh["Content-Type"]; !ok {
		wh.Set("Content-Type", "text/plain; charset=utf-8")
	}
	ctx.ResponseWriter.WriteHeader(status)
	var text string
	if len(a) > 0 {
		text = strings.Join(a, " ")
	} else {
		text = http.StatusText(status)
	}
	ctx.Write([]byte(text))
}

func (ctx *Context) Error(err error) {
	if ctx.mux.Debug {
		ctx.End(http.StatusInternalServerError, err.Error())
	} else {
		if ctx.mux.Logger != nil {
			ctx.mux.Logger.Error(err)
		}
		ctx.End(http.StatusInternalServerError)
	}
}

func (ctx *Context) ServeFile(name string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, name)
}

func (ctx *Context) User() acl.User {
	return ctx.user
}

func (ctx *Context) SetUser(user acl.User) {
	ctx.user = user
}
