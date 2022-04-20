package rex

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
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

// Cors returns a Cors middleware to handle cors.
func Cors(cors CORS) Handle {
	return func(ctx *Context) interface{} {
		if cors.AllowAllOrigins || len(cors.AllowOrigins) > 0 {
			// always set Vary headers
			// see https://github.com/rs/cors/issues/10
			ctx.SetHeader("Vary", "Origin")

			currentOrigin := ctx.R.Header.Get("Origin")
			if currentOrigin == "" {
				return nil
			}

			isPreflight := ctx.R.Method == "OPTIONS"
			allowAll := cors.AllowAllOrigins
			allowCurrent := allowAll
			if !allowAll {
				for _, origin := range cors.AllowOrigins {
					if origin == "*" {
						allowAll = true
						allowCurrent = true
						break
					} else if origin == currentOrigin {
						allowCurrent = true
					}
				}
			}

			if !allowCurrent {
				if isPreflight {
					return http.StatusNoContent
				}
				return nil
			}

			allowOrigin := "*"
			if !allowAll {
				allowOrigin = strings.Join(cors.AllowOrigins, ",")
			}
			if cors.AllowCredentials {
				if allowOrigin == "*" {
					allowOrigin = currentOrigin
				}
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			ctx.SetHeader("Access-Control-Allow-Origin", allowOrigin)

			if isPreflight {
				ctx.SetHeader("Vary", "Access-Control-Request-Method")
				ctx.SetHeader("Vary", "Access-Control-Request-Headers")

				reqMethod := ctx.R.Header.Get("Access-Control-Request-Method")
				if reqMethod == "" {
					// invalid preflight request
					ctx.DeleteHeader("Access-Control-Allow-Origin")
					ctx.DeleteHeader("Access-Control-Allow-Credentials")
					return http.StatusNoContent
				}

				if len(cors.AllowMethods) > 0 {
					ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(cors.AllowMethods, ","))
				}
				if len(cors.AllowHeaders) > 0 {
					ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(cors.AllowHeaders, ","))
				} else {
					reqHeaders := ctx.R.Header.Get("Access-Control-Request-Headers")
					if reqHeaders != "" {
						ctx.SetHeader("Access-Control-Allow-Headers", reqHeaders)
					}
				}
				if cors.MaxAge > 0 {
					ctx.SetHeader("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
				}
				return http.StatusNoContent
			}

			if len(cors.ExposeHeaders) > 0 {
				ctx.SetHeader("Access-Control-Expose-Headers", strings.Join(cors.ExposeHeaders, ","))
			}
		}
		return nil
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
		return 401
	}
}

// AutoCompress is REX middleware to enable compress by content type and client `Accept-Encoding`
func AutoCompress() Handle {
	return func(ctx *Context) interface{} {
		ctx.autoCompress = true
		return nil
	}
}
