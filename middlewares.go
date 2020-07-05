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
	return func(ctx *Context) {
		if key != "" {
			ctx.SetHeader(key, value)
		}
		ctx.Next()
	}
}

// SendError returns a SendError middleware.
func SendError() Handle {
	return func(ctx *Context) {
		ctx.sendError = true
		ctx.Next()
	}
}

// JSONError returns a JSONError middleware to pass error as json.
func JSONError() Handle {
	return func(ctx *Context) {
		ctx.errorType = "json"
		ctx.Next()
	}
}

// ErrorLogger returns a ErrorLogger middleware to sets the error logger.
func ErrorLogger(logger Logger) Handle {
	return func(ctx *Context) {
		if logger != nil {
			ctx.logger = logger
		}
		ctx.Next()
	}
}

// AccessLogger returns a AccessLogger middleware to sets the access logger.
func AccessLogger(logger Logger) Handle {
	return func(ctx *Context) {
		ctx.accessLogger = logger
		ctx.Next()
	}
}

// SIDStore returns a SIDStore middleware to sets sid store for session.
func SIDStore(sidStore session.SIDStore) Handle {
	return func(ctx *Context) {
		if sidStore != nil {
			ctx.sidStore = sidStore
		}
		ctx.Next()
	}
}

// SessionPool returns a SessionPool middleware to set the session pool.
func SessionPool(pool session.Pool) Handle {
	return func(ctx *Context) {
		if pool != nil {
			ctx.sessionPool = pool
		}
		ctx.Next()
	}
}

// Cors returns a Cors middleware to handle cors.
func Cors(cors CORS) Handle {
	return func(ctx *Context) {
		if cors.AllowAllOrigins || len(cors.AllowOrigins) > 0 {
			// always set Vary headers
			// see https://github.com/rs/cors/issues/10
			ctx.SetHeader("Vary", "Origin")

			currentOrigin := ctx.R.Header.Get("Origin")
			if currentOrigin == "" {
				// not a cors resquest
				ctx.Next()
				return
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
					ctx.End(http.StatusNoContent)
				} else {
					ctx.Next()
				}
				return
			}

			allowOrigin := "*"
			if !allowAll {
				allowOrigin = strings.Join(cors.AllowOrigins, ",")
			}
			ctx.SetHeader("Access-Control-Allow-Origin", allowOrigin)
			if cors.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}

			if isPreflight {
				ctx.SetHeader("Vary", "Access-Control-Request-Method")
				ctx.SetHeader("Vary", "Access-Control-Request-Headers")

				reqMethod := ctx.R.Header.Get("Access-Control-Request-Method")
				if reqMethod == "" {
					// invalid preflight request
					ctx.DeleteHeader("Access-Control-Allow-Origin")
					ctx.DeleteHeader("Access-Control-Allow-Credentials")
					ctx.End(http.StatusNoContent)
					return
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
				ctx.End(http.StatusNoContent)
				return
			}

			if len(cors.ExposeHeaders) > 0 {
				ctx.SetHeader("Access-Control-Expose-Headers", strings.Join(cors.ExposeHeaders, ","))
			}
		}
		ctx.Next()
	}
}

// ACL returns a ACL middleware.
func ACL(permissions ...string) Handle {
	return func(ctx *Context) {
		for _, p := range permissions {
			p = strings.TrimSpace(p)
			if p != "" {
				if ctx.acl == nil {
					ctx.acl = map[string]struct{}{}
				}
				ctx.acl[p] = struct{}{}
			}
		}
		ctx.Next()
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware.
func BasicAuth(auth func(name string, password string) (user ACLUser, err error)) Handle {
	return BasicAuthWithRealm("", auth)
}

// BasicAuthWithRealm returns a Basic HTTP Authorization middleware with realm.
func BasicAuthWithRealm(realm string, auth func(name string, password string) (user ACLUser, err error)) Handle {
	return func(ctx *Context) {
		value := ctx.R.Header.Get("Authorization")
		if len(value) > 0 {
			if authType, authData := utils.SplitByFirstByte(value, ' '); len(authData) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(authData)
				if e != nil {
					return
				}

				name, password := utils.SplitByFirstByte(string(authInfo), ':')
				user, err := auth(name, password)
				if err != nil {
					ctx.Error(err.Error(), 500)
					return
				}

				if user != nil {
					ctx.SetACLUser(user)
					ctx.Next()
					return
				}
			}
		}

		if realm == "" {
			realm = "Authorization Required"
		}
		ctx.SetHeader("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
		ctx.W.WriteHeader(401)
	}
}
