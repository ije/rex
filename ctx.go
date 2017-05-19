package webx

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ije/gox/crypto/rs"
	"github.com/ije/gox/utils"
	"github.com/ije/webx/session"
	"github.com/ije/webx/user"
)

type Context struct {
	w       http.ResponseWriter
	r       *http.Request
	host    string
	session session.Session
	user    *user.User
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

func (ctx *Context) SetCookie(name, value string, expires time.Time, httpOnly bool, extra ...string) (cookie *http.Cookie) {
	cookie = &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		Path:     "/",
		Domain:   ctx.host,
		HttpOnly: httpOnly,
	}
	if el := len(extra); el > 0 {
		cookie.Path = extra[0]
		if el > 1 {
			cookie.Domain = extra[1]
		}
	}
	ctx.w.Header().Add("Set-Cookie", cookie.String())
	return
}

func (ctx *Context) RemoveCookie(name string, extra ...string) {
	cookie := &http.Cookie{
		Name:    name,
		Value:   "-",
		Path:    "/",
		Domain:  ctx.host,
		Expires: time.Now().Add(-time.Second),
	}
	if el := len(extra); el > 0 {
		cookie.Path = extra[0]
		if el > 1 {
			cookie.Domain = extra[1]
		}
	}
	ctx.w.Header().Add("Set-Cookie", cookie.String())
	return
}

func (ctx *Context) Session() session.Session {
	if ctx.session != nil {
		return ctx.session
	}

	var sid string
	if c, err := ctx.Cookie("x-session"); err == nil {
		sid = c.Value
	}

	sess, err := xs.Session.Get(sid)
	if err != nil {
		panic("ctx: get session failed: " + err.Error())
	}

	if sid != sess.SID() {
		cookie := &http.Cookie{
			Name:     "x-session",
			Value:    strf("%s:%s", ctx.host, sess.SID),
			Path:     "/",
			Domain:   ctx.host,
			HttpOnly: true,
		}
		ctx.w.Header().Add("Set-Cookie", cookie.String())
	}

	ctx.session = sess
	return sess
}

func (ctx *Context) Logined() bool {
	return ctx.LoginedUser() != nil
}

func (ctx *Context) LoginedUser() *user.User {
	if ctx.user != nil {
		return ctx.user
	}

	if xs.Users == nil {
		return nil
	}

	_, err := ctx.Cookie("x-session")
	if err != nil {
		if _, err = ctx.Cookie("x-token"); err != nil {
			return nil
		}
	}

	if id, ok := ctx.Session().Get("LOGINED_USER"); ok {
		ctx.user, err = xs.Users.Get(id)
		if err != nil {
			panic(strf("ctx.LoginedUser: xs.Users.Get(%d): %v", id, err.Error()))
		}
	}

	if ctx.user == nil {
		if cookie, err := ctx.Cookie("x-token"); err == nil && len(cookie.Value) > 0 {
			user, err := xs.Users.CheckLoginToken(cookie.Value)
			if err != nil {
				panic(strf("ctx.LoginedUser: xs.Users.CheckLoginToken(\"***\"): %v", err))
			}

			if user != nil {
				newToken := rs.Base64.String(64)
				err = xs.Users.UpdateLoginToken(user.ID, newToken)
				if err != nil {
					panic(strf("ctx.LoginedUser: xs.Users.UpdateLoginToken(%d, \"***\"): %v", user.ID, err))
				}

				err = xs.Users.Update(user.ID, map[string]interface{}{"Logined": time.Now()})
				if err != nil {
					panic(strf("ctx.LoginedUser: xs.Users.Update(%d): %v", user.ID, err.Error()))
				}

				ctx.session.Set("USER", user.ID)
				ctx.SetCookie("x-token", newToken, time.Now().Add(7*24*time.Hour), true, "/", ctx.host)
				ctx.user = user
			}
		}
	}

	return ctx.user
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

func (ctx *Context) JSON(status int, data interface{}) (n int, err error) {
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

func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.w.Write(p)
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
					form.Set(key, strf("%v", value))
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

	ctx.w.Header().Set("WWW-Authenticate", strf("Basic realm=\"%s\"", realm))
	ctx.w.WriteHeader(401)
	return
}
