package rex

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
)

// A AclUser interface contains the Permissions method that returns the permission IDs
type AclUser interface {
	Perms() []string
}

// A ILogger interface contains the Printf method.
type ILogger interface {
	Printf(format string, v ...interface{})
}

// A Context to handle http requests.
type Context struct {
	W                http.ResponseWriter
	R                *http.Request
	Store            *Store
	basicAuthUser    string
	aclUser          AclUser
	session          *SessionStub
	sessionPool      session.Pool
	sessionIdHandler session.IdHandler
	compress         bool
	logger           ILogger
	accessLogger     ILogger
}

// Pathname returns the request pathname.
func (ctx *Context) Pathname() string {
	return ctx.R.URL.Path
}

// Query returns the request query values.
func (ctx *Context) Query() url.Values {
	return ctx.R.URL.Query()
}

// SetHeader sets the response header.
func (ctx *Context) SetHeader(key string, value string) {
	ctx.W.Header().Set(key, value)
}

// BasicAuthUser returns the BasicAuth username
func (ctx *Context) BasicAuthUser() string {
	return ctx.basicAuthUser
}

// AclUser returns the ACL user
func (ctx *Context) AclUser() AclUser {
	return ctx.aclUser
}

// Session returns the session if it is undefined then create a new one.
func (ctx *Context) Session() *SessionStub {
	if ctx.sessionPool == nil {
		panic(&invalid{500, "session pool is nil"})
	}

	if ctx.session == nil {
		sid := ctx.sessionIdHandler.Get(ctx.R)
		sess, err := ctx.sessionPool.GetSession(sid)
		if err != nil {
			panic(&invalid{500, err.Error()})
		}

		ctx.session = &SessionStub{sess}

		if sess.SID() != sid {
			ctx.sessionIdHandler.Put(ctx.W, sess.SID())
		}
	}

	return ctx.session
}

// UserAgent returns the request User-Agent.
func (ctx *Context) UserAgent() string {
	return ctx.R.UserAgent()
}

// Cookie returns the cookie by name.
func (ctx *Context) Cookie(name string) (cookie *http.Cookie) {
	cookie, _ = ctx.R.Cookie(name)
	return
}

// SetCookie sets a cookie.
func (ctx *Context) SetCookie(cookie http.Cookie) {
	if cookie.Name != "" {
		ctx.W.Header().Add("Set-Cookie", cookie.String())
	}
}

// RemoveCookie removes the cookie.
func (ctx *Context) RemoveCookie(cookie http.Cookie) {
	if cookie.Name != "" {
		cookie.Value = "-"
		cookie.Expires = time.Unix(0, 0)
		ctx.SetCookie(cookie)
	}
}

// RemoveCookieByName removes the cookie by name.
func (ctx *Context) RemoveCookieByName(name string) {
	ctx.SetCookie(http.Cookie{
		Name:    name,
		Value:   "-",
		Expires: time.Unix(0, 0),
	})
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

func (ctx *Context) enableCompression() {
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
		w, ok := ctx.W.(*rexWriter)
		if ok && !w.headerSent {
			h := w.Header()
			vary := h.Get("Vary")
			if vary == "" {
				h.Set("Vary", "Accept-Encoding")
			} else if !strings.Contains(vary, "Accept-Encoding") {
				h.Set("Vary", fmt.Sprintf("%s, Accept-Encoding", vary))
			}
			h.Set("Content-Encoding", encoding)
			h.Del("Content-Length")
			switch encoding {
			case "br":
				w.compWriter = brotli.NewWriterLevel(w.httpWriter, brotli.BestSpeed)
			case "gzip":
				w.compWriter, _ = gzip.NewWriterLevel(w.httpWriter, gzip.BestSpeed)
			}
		}
	}
}

func (ctx *Context) canBeCompressed(contentType string, contentSize int) bool {
	return ctx.compress && contentSize > 1024 && strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/javascript") || strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "application/xml") || strings.HasPrefix(contentType, "application/wasm")
}

func (ctx *Context) respondWith(v interface{}) {
	s := 0
	status := func() int {
		if s > 0 {
			return s
		}
		return 200
	}
	header := ctx.W.Header()

Switch:
	switch r := v.(type) {
	case http.Handler:
		r.ServeHTTP(ctx.W, ctx.R)

	case *http.Response:
		for k, v := range r.Header {
			header[k] = v
		}
		ctx.W.WriteHeader(r.StatusCode)
		io.Copy(ctx.W, r.Body)
		r.Body.Close()

	case *redirect:
		header.Set("Location", hexEscapeNonASCII(r.url))
		ctx.W.WriteHeader(r.status)

	case string:
		data := []byte(r)
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "text/plain; charset=utf-8")
		}
		if ctx.compress && len(data) > 1024 {
			ctx.enableCompression()
		} else {
			header.Set("Content-Length", strconv.Itoa(len(data)))
		}
		ctx.W.WriteHeader(status())
		ctx.W.Write(data)

	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		header.Set("Content-Type", "text/plain")
		ctx.W.WriteHeader(status())
		fmt.Fprintf(ctx.W, "%v", r)

	case []byte:
		cType := header.Get("Content-Type")
		if ctx.canBeCompressed(cType, len(r)) {
			ctx.enableCompression()
		} else {
			header.Set("Content-Length", strconv.Itoa(len(r)))
		}
		if cType == "" {
			header.Set("Content-Type", "binary/octet-stream")
		}
		ctx.W.WriteHeader(status())
		ctx.W.Write(r)

	case io.Reader:
		defer func() {
			if c, ok := r.(io.Closer); ok {
				c.Close()
			}
		}()
		size := 0
		if s, ok := r.(io.Seeker); ok {
			n, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				ctx.respondWithError(&Error{500, err.Error()})
				return
			}
			_, err = s.Seek(0, io.SeekStart)
			if err != nil {
				ctx.respondWithError(&Error{500, err.Error()})
				return
			}
			size = int(n)
		}
		cType := header.Get("Content-Type")
		if ctx.canBeCompressed(cType, size) {
			ctx.enableCompression()
		} else if size > 0 {
			header.Set("Content-Length", strconv.Itoa(size))
		}
		if cType == "" {
			header.Set("Content-Type", "binary/octet-stream")
		}
		ctx.W.WriteHeader(status())
		io.Copy(ctx.W, r)

	case *content:
		defer func() {
			if c, ok := r.content.(io.Closer); ok {
				c.Close()
			}
		}()
		if ctx.compress {
			isTextContent := false
			switch strings.TrimPrefix(path.Ext(r.name), ".") {
			case "html", "htm", "xml", "svg", "css", "less", "sass", "scss", "json", "json5", "map", "js", "jsx", "mjs", "cjs", "ts", "mts", "tsx", "md", "mdx", "yaml", "txt", "wasm":
				isTextContent = true
			}
			if isTextContent {
				size, err := r.content.Seek(0, io.SeekEnd)
				if err != nil {
					ctx.respondWithError(&Error{500, err.Error()})
					return
				}
				_, err = r.content.Seek(0, io.SeekStart)
				if err != nil {
					ctx.respondWithError(&Error{500, err.Error()})
					return
				}
				if size > 1024 {
					ctx.enableCompression()
				}
			}
		}
		if r.mtime.IsZero() {
			r.mtime = time.Now()
			if header.Get("Cache-Control") == "" {
				header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			}
		}
		http.ServeContent(ctx.W, ctx.R, r.name, r.mtime, r.content)

	case *statusd:
		s = r.code
		v = r.content
		goto Switch

	case *fs:
		filepath := path.Join(r.root, ctx.R.URL.Path)
		fi, err := os.Stat(filepath)
		if err == nil && fi.IsDir() {
			filepath = path.Join(filepath, "index.html")
			_, err = os.Stat(filepath)
		}
		if err != nil && os.IsNotExist(err) && r.fallback != "" {
			filepath = path.Join(r.root, utils.CleanPath(r.fallback))
			_, err = os.Stat(filepath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				ctx.respondWithError(&Error{404, "not found"})
			} else {
				ctx.respondWithError(&Error{500, err.Error()})
			}
			return
		}
		v = File(filepath)
		goto Switch

	case error:
		if s >= 400 && s < 600 {
			ctx.respondWithError(&Error{s, r.Error()})
		} else {
			ctx.respondWithError(&Error{500, r.Error()})
		}

	case *Error:
		ctx.respondWithError(r)

	case Error:
		ctx.respondWithError(&r)

	default:
		ctx.respondWithJson(r, status())
	}
}

func (ctx *Context) respondWithJson(v interface{}, status int) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(v)
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		ctx.W.WriteHeader(500)
		ctx.W.Write([]byte(`{"error": {"status": 500, "message": "bad json"}}`))
		return
	}
	if ctx.compress && buf.Len() > 1024 {
		ctx.enableCompression()
	} else {
		ctx.W.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	}
	ctx.W.WriteHeader(status)
	io.Copy(ctx.W, buf)
}

func (ctx *Context) respondWithError(err *Error) {
	if err.Status >= 500 && ctx.logger != nil {
		ctx.logger.Printf("[error] %s", err.Message)
	}
	ctx.respondWithJson(map[string]interface{}{
		"error": err,
	}, err.Status)
}

func hexEscapeNonASCII(s string) string {
	newLen := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			newLen += 3
		} else {
			newLen++
		}
	}
	if newLen == len(s) {
		return s
	}
	b := make([]byte, 0, newLen)
	var pos int
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			if pos < i {
				b = append(b, s[pos:i]...)
			}
			b = append(b, '%')
			b = strconv.AppendInt(b, int64(s[i]), 16)
			pos = i + 1
		}
	}
	if pos < len(s) {
		b = append(b, s[pos:]...)
	}
	return string(b)
}
