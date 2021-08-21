package rex

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
)

// A Context to handle http requests.
type Context struct {
	W             http.ResponseWriter
	R             *http.Request
	Path          *Path
	Form          *Form
	Store         *Store
	basicAuthUser string
	acl           map[string]struct{}
	aclUser       ACLUser
	session       *Session
	sessionPool   session.Pool
	sidStore      session.SIDStore
	logger        Logger
	accessLogger  Logger
}

// BasicAuthUser returns the BasicAuth username
func (ctx *Context) BasicAuthUser() string {
	return ctx.basicAuthUser
}

// ACLUser returns the acl user
func (ctx *Context) ACLUser() ACLUser {
	return ctx.aclUser
}

// SetACLUser sets the acl user
func (ctx *Context) SetACLUser(user ACLUser) {
	ctx.aclUser = user
}

// Session returns the session if it is undefined then create a new one.
func (ctx *Context) Session() *Session {
	if ctx.sessionPool == nil {
		panic(&recoverError{500, "session pool is nil"})
	}

	if ctx.session == nil {
		sid := ctx.sidStore.Get(ctx.R)
		sess, err := ctx.sessionPool.GetSession(sid)
		if err != nil {
			panic(&recoverError{500, err.Error()})
		}

		ctx.session = &Session{sess}

		if sess.SID() != sid {
			ctx.sidStore.Put(ctx.W, sess.SID())
		}
	}

	return ctx.session
}

// Cookie returns the cookie by name.
func (ctx *Context) Cookie(name string) (cookie *http.Cookie, err error) {
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

// DeleteHeader deletes the values associated with key.
func (ctx *Context) DeleteHeader(key string) {
	ctx.W.Header().Del(key)
}

// RemoteIP returns the remote client IP.
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

// EnableCompression enables the compression method based on the Accept-Encoding header
func (ctx *Context) EnableCompression() {
	var encoding string
	for _, p := range strings.Split(ctx.R.Header.Get("Accept-Encoding"), ",") {
		name, _ := utils.SplitByFirstByte(p, ';')
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "br":
			encoding = "br"
		case "gzip":
			if encoding == "" {
				encoding = "gzip"
			}
		}
	}
	if encoding != "" {
		w, ok := ctx.W.(*responseWriter)
		if ok && !w.headerSent {
			h := w.Header()
			if h.Get("Vary") == "" {
				h.Set("Vary", "Accept-Encoding")
			}
			if h.Get("Content-Length") != "" {
				h.Del("Content-Length")
			}
			h.Set("Content-Encoding", encoding)
			switch encoding {
			case "br":
				w.compression = brotli.NewWriterLevel(w.rawWriter, brotli.BestSpeed)
			case "gzip":
				w.compression, _ = gzip.NewWriterLevel(w.rawWriter, gzip.BestSpeed)
			}
		}
	}
}

func (ctx *Context) end(v interface{}, args ...int) {
	status := 0
	if len(args) > 0 {
		status = args[0]
	}

	switch r := v.(type) {
	case *redirect:
		http.Redirect(ctx.W, ctx.R, r.url, r.status)

	case string:
		if ctx.W.Header().Get("Content-Type") == "" {
			ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
		}
		if len(r) > 1024 {
			ctx.EnableCompression()
		}
		if status >= 100 {
			ctx.W.WriteHeader(status)
		}
		ctx.W.Write([]byte(r))

	case []byte:
		if ctx.W.Header().Get("Content-Type") == "" {
			ctx.SetHeader("Content-Type", "application/octet-stream")
		}
		if status >= 100 {
			ctx.W.WriteHeader(status)
		}
		ctx.W.Write(r)

	case io.Reader:
		if ctx.W.Header().Get("Content-Type") == "" {
			ctx.SetHeader("Content-Type", "application/octet-stream")
		}
		if status >= 100 {
			ctx.W.WriteHeader(status)
		}
		io.Copy(ctx.W, r)

	case *contentful:
		compressable := false
		switch strings.TrimPrefix(path.Ext(r.name), ".") {
		case "html", "htm", "xml", "svg", "css", "less", "sass", "scss", "json", "json5", "map", "js", "jsx", "mjs", "cjs", "ts", "tsx", "md", "mdx", "yaml", "txt", "wasm":
			compressable = true
		}
		size, err := r.content.Seek(0, io.SeekEnd)
		if err != nil {
			ctx.ejson(&Error{500, err.Error()})
			return
		}
		_, err = r.content.Seek(0, io.SeekStart)
		if err != nil {
			ctx.ejson(&Error{500, err.Error()})
			return
		}
		if compressable && size > 1024 {
			ctx.EnableCompression()
		}
		http.ServeContent(ctx.W, ctx.R, r.name, r.mtime, r.content)
		c, ok := r.content.(io.Closer)
		if ok {
			c.Close()
		}

	case *statusPlayload:
		if status >= 100 {
			ctx.end(r.payload, status)
		} else {
			ctx.end(r.payload, r.status)
		}

	case *fs:
		filepath := path.Join(r.root, ctx.Path.String())
		fi, err := os.Stat(filepath)
		if err == nil && fi.IsDir() {
			filepath = path.Join(filepath, "index.html")
			fi, err = os.Stat(filepath)
		}
		if err != nil && os.IsNotExist(err) && r.fallback != "" {
			filepath = path.Join(r.root, utils.CleanPath(r.fallback))
			fi, err = os.Stat(filepath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				ctx.ejson(&Error{404, "not found"})
			} else {
				ctx.ejson(&Error{500, err.Error()})
			}
			return
		}
		ctx.end(File(filepath))

	case error:
		if status >= 100 {
			ctx.ejson(&Error{status, r.Error()})
		} else {
			ctx.ejson(&Error{500, r.Error()})
		}

	default:
		_, err := utils.ToNumber(r)
		if err == nil {
			ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
			if status >= 100 {
				ctx.W.WriteHeader(status)
			}
			fmt.Fprintf(ctx.W, "%v", r)
			return
		}

		switch e := r.(type) {
		case *Error:
			ctx.ejson(e)
			return
		case Error:
			ctx.ejson(&e)
			return
		}

		ctx.json(r, status)
	}
}

func (ctx *Context) ejson(err *Error) {
	if err.Status >= 500 && ctx.logger != nil {
		ctx.logger.Printf("[error] %s", err.Message)
	}
	ctx.json(map[string]interface{}{
		"error": err,
	}, err.Status)
}

func (ctx *Context) json(v interface{}, status int) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(v)
	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		ctx.W.WriteHeader(500)
		ctx.W.Write([]byte(`{"error": {"status": 500, "message": "bad json"}}`))
		return
	}
	if buf.Len() > 1024 {
		ctx.EnableCompression()
	}
	if status >= 100 {
		ctx.W.WriteHeader(status)
	}
	io.Copy(ctx.W, buf)
}
