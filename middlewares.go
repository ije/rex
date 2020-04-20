package rex

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ije/gox/utils"
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

// Config returns a Config middleware.
func Config(config *Conf) Handle {
	return func(ctx *Context) {
		if config.SendError {
			ctx.sendError = true
		}
		if config.ErrorType != "" {
			ctx.errorType = config.ErrorType
		}
		if config.Logger != nil {
			ctx.logger = config.Logger
		}
		if config.AccessLogger != nil {
			ctx.accessLogger = config.AccessLogger
		}
		if config.SIDStore != nil {
			ctx.sidStore = config.SIDStore
		}
		if config.SessionPool != nil {
			ctx.sessionPool = config.SessionPool
		}
		if config.CORS != nil {
			cors := config.CORS
			isPreflight := ctx.R.Method == "OPTIONS"
			if len(cors.AllowOrigin) > 0 {
				ctx.SetHeader("Access-Control-Allow-Origin", cors.AllowOrigin)
				if cors.AllowCredentials {
					ctx.SetHeader("Access-Control-Allow-Credentials", "true")
				}
				if len(cors.ExposeHeaders) > 0 {
					ctx.SetHeader("Access-Control-Expose-Headers", strings.Join(cors.ExposeHeaders, ", "))
				}
				if isPreflight {
					if len(cors.AllowMethods) > 0 {
						ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(cors.AllowMethods, ", "))
					}
					if len(cors.AllowHeaders) > 0 {
						ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(cors.AllowHeaders, ", "))
					}
					if cors.MaxAge > 0 {
						ctx.SetHeader("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
					}
					ctx.End(http.StatusNoContent)
					return
				}
			} else {
				ctx.AddHeader("Vary", "Origin")
				if isPreflight {
					ctx.AddHeader("Vary", "Access-Control-Request-Method")
					ctx.AddHeader("Vary", "Access-Control-Request-Headers")
				}
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
				if ctx.permissions == nil {
					ctx.permissions = map[string]struct{}{}
				}
				ctx.permissions[p] = struct{}{}
			}
		}
		ctx.Next()
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware.
func BasicAuth(authFn func(name string, password string) (ok bool, err error)) Handle {
	return BasicAuthWithRealm("", authFn)
}

// BasicAuthWithRealm returns a Basic HTTP Authorization middleware with realm.
func BasicAuthWithRealm(realm string, authFn func(name string, password string) (ok bool, err error)) Handle {
	return func(ctx *Context) {
		if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
			if authType, authData := utils.SplitByFirstByte(auth, ' '); len(authData) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(authData)
				if e != nil {
					return
				}

				name, password := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err := authFn(name, password)
				if err != nil {
					ctx.Error(err.Error(), 500)
					return
				}

				if ok {
					ctx.basicUser = BasicUser{
						Name:     name,
						Password: password,
					}
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
