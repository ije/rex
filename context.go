package rex

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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
	R                *http.Request
	W                http.ResponseWriter
	queryRaw         string
	query            url.Values
	header           http.Header
	basicAuthUser    string
	aclUser          AclUser
	session          *SessionStub
	sessionPool      session.Pool
	sessionIdHandler session.SidHandler
	logger           ILogger
	accessLogger     ILogger
	compress         bool
}

// Next executes the next middleware in the chain.
func (ctx *Context) Next() any {
	return next
}

// Method returns the request method.
func (ctx *Context) Method() string {
	return ctx.R.Method
}

// Pathname returns the request pathname.
func (ctx *Context) Pathname() string {
	return ctx.R.URL.Path
}

// PathValue returns the value for the named path wildcard in the [ServeMux] pattern
// that matched the request.
// It returns the empty string if the request was not matched against a pattern
// or there is no such wildcard in the pattern.
func (ctx *Context) PathValue(key string) string {
	return ctx.R.PathValue(key)
}

// RawQuery returns the request raw query string.
func (ctx *Context) RawQuery() string {
	return ctx.R.URL.RawQuery
}

// Query parses RawQuery and returns the corresponding values. It silently discards malformed value pairs.
func (ctx *Context) Query() url.Values {
	if url := ctx.R.URL; ctx.query == nil || ctx.queryRaw != url.RawQuery {
		ctx.queryRaw = url.RawQuery
		ctx.query = url.Query()
	}
	return ctx.query
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
		panic(&invalid{500, "session pool is not set"})
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

// Set sets the header entries associated with key to the
// single element value. It replaces any existing values
// associated with key. The key is case insensitive; it is
// canonicalized by [textproto.CanonicalMIMEHeaderKey].
// To use non-canonical keys, assign to the map directly.
func (ctx *Context) SetHeader(key, value string) {
	ctx.header.Set(key, value)
}

// Cookie returns the named cookie provided in the request or
// [ErrNoCookie] if not found.
// If multiple cookies match the given name, only one cookie will
// be returned.
func (ctx *Context) Cookie(name string) (cookie *http.Cookie) {
	cookie, _ = ctx.R.Cookie(name)
	return
}

// SetCookie sets a cookie to the response.
func (ctx *Context) SetCookie(cookie http.Cookie) {
	if cookie.Name != "" {
		ctx.header.Add("Set-Cookie", cookie.String())
	}
}

// DeleteCookie sets a cookie to the response with an expiration time in the past.
func (ctx *Context) DeleteCookie(cookie http.Cookie) {
	cookie.Value = "-"
	cookie.Expires = time.Unix(0, 0)
	ctx.SetCookie(cookie)
}

// DeleteCookieByName sets a cookie to the response with an expiration time in the past.
func (ctx *Context) DeleteCookieByName(name string) {
	ctx.SetCookie(http.Cookie{
		Name:    name,
		Value:   "-",
		Expires: time.Unix(0, 0),
	})
}

// FormValue returns the first value for the named component of the query.
// The precedence order:
//  1. application/x-www-form-urlencoded form body (POST, PUT, PATCH only)
//  2. query parameters (always)
//  3. multipart/form-data form body (always)
//
// FormValue calls [Request.ParseMultipartForm] and [Request.ParseForm]
// if necessary and ignores any errors returned by these functions.
// If key is not present, FormValue returns the empty string.
// To access multiple values of the same key, call ParseForm and
// then inspect [Request.Form] directly.
func (ctx *Context) FormValue(key string) string {
	return ctx.R.FormValue(key)
}

// PostFormValue returns the first value for the named component of the POST,
// PUT, or PATCH request body. URL query parameters are ignored.
// PostFormValue calls [Request.ParseMultipartForm] and [Request.ParseForm] if necessary and ignores
// any errors returned by these functions.
// If key is not present, PostFormValue returns the empty string.
func (ctx *Context) PostFormValue(key string) string {
	return ctx.R.PostFormValue(key)
}

// FormFile returns the first file for the provided form key.
// FormFile calls [Request.ParseMultipartForm] and [Request.ParseForm] if necessary.
func (ctx *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.R.FormFile(key)
}

func (ctx *Context) enableCompression() bool {
	var encoding string
	accectEncoding := ctx.R.Header.Get("Accept-Encoding")
	if accectEncoding != "" && strings.Contains(accectEncoding, "br") {
		encoding = "br"
	} else if accectEncoding != "" && strings.Contains(accectEncoding, "gzip") {
		encoding = "gzip"
	}
	if encoding != "" {
		w, ok := ctx.W.(*rexWriter)
		if ok {
			h := w.Header()
			if v := h.Get("Vary"); v == "" {
				h.Set("Vary", "Accept-Encoding")
			} else if !strings.Contains(v, "Accept-Encoding") && !strings.Contains(v, "accept-encoding") {
				h.Set("Vary", v+", Accept-Encoding")
			}
			h.Set("Content-Encoding", encoding)
			h.Del("Content-Length")
			switch encoding {
			case "br":
				w.zWriter = brotli.NewWriterLevel(w.rawWriter, brotli.BestSpeed)
			case "gzip":
				w.zWriter, _ = gzip.NewWriterLevel(w.rawWriter, gzip.BestSpeed)
			}
			return true
		}
	}
	return false
}

func (ctx *Context) respondWith(v any) {
	w := ctx.W
	h := w.Header()
	code := 200

SWITCH:
	switch r := v.(type) {
	case http.Handler:
		r.ServeHTTP(w, ctx.R)

	case *http.Response:
		for k, v := range r.Header {
			h[k] = v
		}
		w.WriteHeader(r.StatusCode)
		io.Copy(w, r.Body)
		r.Body.Close()

	case *redirect:
		h.Set("Location", hexEscapeNonASCII(r.url))
		w.WriteHeader(r.status)

	case string:
		data := []byte(r)
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", "text/plain; charset=utf-8")
		}
		if ctx.compress {
			if len(data) < compressMinSize || !ctx.enableCompression() {
				h.Set("Content-Length", strconv.Itoa(len(data)))
			}
		} else {
			h.Set("Content-Length", strconv.Itoa(len(data)))
		}
		w.WriteHeader(code)
		w.Write(data)

	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", "text/plain")
		}
		w.WriteHeader(code)
		fmt.Fprintf(w, "%v", r)

	case []byte:
		cType := h.Get("Content-Type")
		if ctx.compress && isTextContent(cType) {
			if len(r) < compressMinSize || !ctx.enableCompression() {
				h.Set("Content-Length", strconv.Itoa(len(r)))
			}
		} else {
			h.Set("Content-Length", strconv.Itoa(len(r)))
		}
		if cType == "" {
			h.Set("Content-Type", "binary/octet-stream")
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
			if err == nil {
				_, err = s.Seek(0, io.SeekStart)
				if err != nil {
					ctx.respondWithError(err)
					return
				}
				size = int(n)
			}
		}
		cType := h.Get("Content-Type")
		if cType == "" {
			h.Set("Content-Type", "binary/octet-stream")
		}
		if ctx.compress && isTextContent(cType) {
			if size >= 0 {
				if size < compressMinSize || !ctx.enableCompression() {
					h.Set("Content-Length", strconv.Itoa(size))
				}
			} else {
				// unable to seek, compress the content anyway
				ctx.enableCompression()
			}
		} else if size >= 0 {
			h.Set("Content-Length", strconv.Itoa(size))
		}
		w.WriteHeader(code)
		io.Copy(w, r)

	case *content:
		if c, ok := r.content.(io.Closer); ok {
			defer c.Close()
		}
		size := -1
		if s, ok := r.content.(io.Seeker); ok {
			n, err := s.Seek(0, io.SeekEnd)
			if err == nil {
				_, err = s.Seek(0, io.SeekStart)
				if err != nil {
					ctx.respondWithError(err)
					return
				}
				size = int(n)
			}
		}
		if ctx.compress && isTextFile(r.name) {
			if size >= 0 {
				if size < compressMinSize || !ctx.enableCompression() {
					h.Set("Content-Length", strconv.Itoa(int(size)))
				}
			} else {
				// unable to seek, compress the content anyway
				ctx.enableCompression()
			}
		} else if size >= 0 {
			h.Set("Content-Length", strconv.Itoa(size))
		}
		etag := h.Get("ETag")
		if etag != "" && etag == ctx.R.Header.Get("If-None-Match") {
			w.WriteHeader(304)
			return
		}
		if r.mtime.IsZero() {
			if h.Get("Cache-Control") == "" {
				h.Set("Cache-Control", "public, max-age=0, must-revalidate")
			}
		} else {
			if checkIfModifiedSince(ctx.R, r.mtime) {
				w.WriteHeader(304)
				return
			}
			h.Set("Last-Modified", r.mtime.UTC().Format(http.TimeFormat))
		}
		ctype := h.Get("Content-Type")
		if ctype == "" {
			ctype = mime.TypeByExtension(path.Ext(r.name))
			if ctype != "" {
				h.Set("Content-Type", ctype)
			}
		}
		w.WriteHeader(code)
		if ctx.R.Method != "HEAD" {
			io.Copy(w, r.content)
		}

	case *noContent:
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
				w.WriteHeader(404)
				w.Write([]byte("Not Found"))
			} else {
				ctx.respondWithError(err)
			}
			return
		}
		v = File(filepath)
		goto SWITCH

	case *invalid:
		h.Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(r.code)
		w.Write([]byte(r.message))

	case Error:
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(r.Code)
		json.NewEncoder(w).Encode(r)

	case *Error:
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(r.Code)
		json.NewEncoder(w).Encode(r)

	case error:
		ctx.respondWithError(r)

	default:
		buf := bytes.NewBuffer(nil)
		err := json.NewEncoder(buf).Encode(v)
		h.Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error": {"status": 500, "message": "bad json"}}`))
			return
		}
		if !ctx.compress || buf.Len() < compressMinSize || !ctx.enableCompression() {
			h.Set("Content-Length", strconv.Itoa(buf.Len()))
		}
		w.WriteHeader(code)
		io.Copy(w, buf)
	}
}

func (ctx *Context) respondWithError(err error) {
	w := ctx.W
	message := err.Error()
	if ctx.logger != nil {
		ctx.logger.Printf("[error] %s", message)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(500)
	w.Write([]byte(message))
}

func isTextFile(filename string) bool {
	switch strings.TrimPrefix(path.Ext(filename), ".") {
	case "html", "htm", "xml", "svg", "css", "less", "sass", "scss", "json", "json5", "map", "js", "jsx", "mjs", "cjs", "ts", "mts", "tsx", "md", "mdx", "yaml", "txt", "wasm":
		return true
	default:
		return false
	}
}

func isTextContent(contentType string) bool {
	return contentType != "" && (strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/javascript") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/wasm"))
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
