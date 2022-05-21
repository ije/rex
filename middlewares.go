package rex

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
	"github.com/rs/cors"
)

// Header is REX middleware to set http header
func Header(key string, value string) Handle {
	return func(ctx *Context) interface{} {
		if key != "" {
			ctx.SetHeader(key, value)
		}
		return nil
	}
}

// ErrorLogger returns a ErrorLogger middleware to sets the error logger.
func ErrorLogger(logger Logger) Handle {
	return func(ctx *Context) interface{} {
		if logger != nil {
			ctx.logger = logger
		}
		return nil
	}
}

// AccessLogger returns a AccessLogger middleware to sets the access logger.
func AccessLogger(logger Logger) Handle {
	return func(ctx *Context) interface{} {
		ctx.accessLogger = logger
		return nil
	}
}

// SIDStore returns a SIDStore middleware to sets sid store for session.
func SIDStore(sidStore session.SIDStore) Handle {
	return func(ctx *Context) interface{} {
		if sidStore != nil {
			ctx.sidStore = sidStore
		}
		return nil
	}
}

// SessionPool returns a SessionPool middleware to set the session pool.
func SessionPool(pool session.Pool) Handle {
	return func(ctx *Context) interface{} {
		if pool != nil {
			ctx.sessionPool = pool
		}
		return nil
	}
}

// Cors returns a Cors middleware to handle CORS.
func Cors(cors *cors.Cors) Handle {
	return func(ctx *Context) interface{} {
		optionPassthrough := false
		h := func(w http.ResponseWriter, r *http.Request) {
			optionPassthrough = true
		}
		cors.ServeHTTP(ctx.W, ctx.R, h)
		if optionPassthrough {
			return nil // next
		}
		return h // end
	}
}

// ACL returns a ACL middleware.
func ACL(permissions ...string) Handle {
	return func(ctx *Context) interface{} {
		for _, p := range permissions {
			p = strings.TrimSpace(p)
			if p != "" {
				if ctx.acl == nil {
					ctx.acl = map[string]struct{}{}
				}
				ctx.acl[p] = struct{}{}
			}
		}
		return nil
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware.
func BasicAuth(auth func(name string, secret string) (ok bool, err error)) Handle {
	return BasicAuthWithRealm("", auth)
}

// BasicAuthWithRealm returns a Basic HTTP Authorization middleware with realm.
func BasicAuthWithRealm(realm string, auth func(name string, secret string) (ok bool, err error)) Handle {
	return func(ctx *Context) interface{} {
		value := ctx.R.Header.Get("Authorization")
		if strings.HasPrefix(value, "Basic ") {
			authInfo, err := base64.StdEncoding.DecodeString(value[6:])
			if err == nil {
				name, secret := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err := auth(name, secret)
				if err != nil {
					return &Error{500, err.Error()}
				}
				if ok {
					ctx.basicAuthUser = name
					return nil
				}
			}
		}

		if realm == "" {
			realm = "Authorization Required"
		}
		ctx.SetHeader("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
		return Status(401, "")
	}
}

// Compression is REX middleware to enable compress by content type and client `Accept-Encoding`
func Compression() Handle {
	return func(ctx *Context) interface{} {
		ctx.compression = true
		return nil
	}
}
