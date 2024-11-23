package rex

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
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
	Printf(format string, v ...any)
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

// GetHeader returns the request header by key.
func (ctx *Context) GetHeader(key string) string {
	return ctx.R.Header.Get(key)
}

// SetHeader sets the response header.
func (ctx *Context) SetHeader(key string, value string) {
	ctx.W.(*rexWriter).header.Set(key, value)
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
	return ctx.R.Header.Get("User-Agent")
}

// Cookie returns the request cookie by name.
func (ctx *Context) Cookie(name string) (cookie *http.Cookie) {
	cookie, _ = ctx.R.Cookie(name)
	return
}

// SetCookie sets a cookie to the response.
func (ctx *Context) SetCookie(cookie http.Cookie) {
	if cookie.Name != "" {
		ctx.W.Header().Add("Set-Cookie", cookie.String())
	}
}

// ClearCookie sets a cookie to the response with an expiration time in the past.
func (ctx *Context) ClearCookie(cookie http.Cookie) {
	if cookie.Name != "" {
		cookie.Value = "-"
		cookie.Expires = time.Unix(0, 0)
		ctx.SetCookie(cookie)
	}
}

// ClearCookieByName sets a cookie to the response with an expiration time in the past.
func (ctx *Context) ClearCookieByName(name string) {
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

func (ctx *Context) setCompressWriter() {
	var encoding string
	if accectEncoding := ctx.R.Header.Get("Accept-Encoding"); accectEncoding != "" && strings.Contains(accectEncoding, "br") {
		encoding = "br"
	} else if accectEncoding != "" && strings.Contains(accectEncoding, "gzip") {
		encoding = "gzip"
	}
	if encoding != "" {
		w, ok := ctx.W.(*rexWriter)
		if ok {
			if !w.headerSent {
				h := w.Header()
				vary := h.Get("Vary")
				if vary == "" {
					h.Set("Vary", "Accept-Encoding")
				} else if !strings.Contains(vary, "Accept-Encoding") {
					h.Set("Vary", fmt.Sprintf("%s, Accept-Encoding", vary))
				}
				h.Set("Content-Encoding", encoding)
				h.Del("Content-Length")
			}
			if encoding == "br" {
				w.compWriter = brotli.NewWriterLevel(w.httpWriter, brotli.BestSpeed)
			} else if encoding == "gzip" {
				w.compWriter, _ = gzip.NewWriterLevel(w.httpWriter, gzip.BestSpeed)
			}
		}
	}
}

func (ctx *Context) isCompressible(contentType string, contentSize int) bool {
	return ctx.compress && contentSize > 1024 && contentType != "" && (strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/javascript") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/wasm"))
}

func (ctx *Context) respondWith(v any) {
	w := ctx.W
	header := w.(*rexWriter).header
	code := 200

SWITCH:
	switch r := v.(type) {
	case http.Handler:
		r.ServeHTTP(w, ctx.R)

	case *http.Response:
		for k, v := range r.Header {
			header[k] = v
		}
		w.WriteHeader(r.StatusCode)
		io.Copy(w, r.Body)
		r.Body.Close()

	case *redirect:
		header.Set("Location", hexEscapeNonASCII(r.url))
		w.WriteHeader(r.status)

	case string:
		data := []byte(r)
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "text/plain; charset=utf-8")
		}
		if ctx.compress && len(data) > 1024 {
			ctx.setCompressWriter()
		} else {
			header.Set("Content-Length", strconv.Itoa(len(data)))
		}
		w.WriteHeader(code)
		w.Write(data)

	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "text/plain")
		}
		w.WriteHeader(code)
		fmt.Fprintf(w, "%v", r)

	case []byte:
		cType := header.Get("Content-Type")
		if ctx.isCompressible(cType, len(r)) {
			ctx.setCompressWriter()
		} else {
			header.Set("Content-Length", strconv.Itoa(len(r)))
		}
		if cType == "" {
			header.Set("Content-Type", "binary/octet-stream")
		}
		w.WriteHeader(code)
		w.Write(r)

	case io.Reader:
		defer func() {
			if c, ok := r.(io.Closer); ok {
				c.Close()
			}
		}()
		size := -1
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
		if cType == "" {
			header.Set("Content-Type", "binary/octet-stream")
		}
		if size >= 0 {
			if ctx.isCompressible(cType, size) {
				ctx.setCompressWriter()
			} else {
				header.Set("Content-Length", strconv.Itoa(size))
			}
		}
		w.WriteHeader(code)
		io.Copy(w, r)

	case *content:
		defer func() {
			if c, ok := r.content.(io.Closer); ok {
				c.Close()
			}
		}()
		if ctx.compress {
			isText := false
			switch strings.TrimPrefix(path.Ext(r.name), ".") {
			case "html", "htm", "xml", "svg", "css", "less", "sass", "scss", "json", "json5", "map", "js", "jsx", "mjs", "cjs", "ts", "mts", "tsx", "md", "mdx", "yaml", "txt", "wasm":
				isText = true
			}
			if isText {
				if seeker, ok := r.content.(io.Seeker); ok {
					size, err := seeker.Seek(0, io.SeekEnd)
					if err != nil {
						ctx.respondWithError(&Error{500, err.Error()})
						return
					}
					_, err = seeker.Seek(0, io.SeekStart)
					if err != nil {
						ctx.respondWithError(&Error{500, err.Error()})
						return
					}
					if size > 1024 {
						ctx.setCompressWriter()
					}
				} else {
					// unable to seek, so compress it anyway
					ctx.setCompressWriter()
				}
			}
		}
		if r.mtime.IsZero() {
			r.mtime = time.Now()
			if header.Get("Cache-Control") == "" {
				header.Set("Cache-Control", "public, max-age=0, must-revalidate")
			}
		}
		if readSeeker, ok := r.content.(io.ReadSeeker); ok {
			http.ServeContent(w, ctx.R, r.name, r.mtime, readSeeker)
		} else {
			if checkIfModifiedSince(ctx.R, r.mtime) {
				w.WriteHeader(304)
				return
			}
			h := w.Header()
			ctype := h.Get("Content-Type")
			if ctype == "" {
				ctype = mime.TypeByExtension(path.Ext(r.name))
				if ctype != "" {
					h.Set("Content-Type", ctype)
				}
			}
			if ctx.R.Method != "HEAD" {
				io.Copy(w, r.content)
			}
		}

	case *nocontent:
		w.WriteHeader(http.StatusNoContent)

	case *status:
		code = r.code
		v = r.content
		if v == nil {
			w.WriteHeader(code)
		} else {
			goto SWITCH
		}

	case *fs:
		filepath := path.Join(r.root, ctx.R.URL.Path)
		fi, err := os.Stat(filepath)
		if err == nil && fi.IsDir() {
			filepath = path.Join(filepath, "index.html")
			_, err = os.Stat(filepath)
		}
		if err != nil && os.IsNotExist(err) && r.fallback != "" {
			filepath = path.Join(r.root, r.fallback)
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
		goto SWITCH

	case error:
		if code >= 400 && code < 600 {
			ctx.respondWithError(&Error{code, r.Error()})
		} else {
			ctx.respondWithError(&Error{500, r.Error()})
		}

	case Error:
		ctx.respondWithError(&r)

	case *Error:
		ctx.respondWithError(r)

	default:
		ctx.respondWithJson(v, code)
	}
}

func (ctx *Context) respondWithJson(v any, status int) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(v)
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		ctx.W.WriteHeader(500)
		ctx.W.Write([]byte(`{"error": {"status": 500, "message": "bad json"}}`))
		return
	}
	if ctx.compress && buf.Len() > 1024 {
		ctx.setCompressWriter()
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
	ctx.respondWithJson(map[string]any{
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

func checkIfModifiedSince(r *http.Request, modtime time.Time) bool {
	if r.Method != "GET" && r.Method != "HEAD" {
		return false
	}
	ims := r.Header.Get("If-Modified-Since")
	if ims == "" || modtime.IsZero() || modtime.Equal(time.Unix(0, 0)) {
		return false
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return false
	}
	// The Last-Modified header truncates sub-second precision so
	// the modtime needs to be truncated too.
	modtime = modtime.Truncate(time.Second)
	return modtime.Compare(t) > 0
}
