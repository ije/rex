package rex

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path"
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

// CORS returns a CORS middleware.
func CORS(opts CORSOptions) Handle {
	return func(ctx *Context) {
		isPreflight := ctx.R.Method == "OPTIONS"
		if len(opts.AllowOrigin) > 0 {
			ctx.SetHeader("Access-Control-Allow-Origin", opts.AllowOrigin)
			if opts.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if len(opts.ExposeHeaders) > 0 {
				ctx.SetHeader("Access-Control-Expose-Headers", strings.Join(opts.ExposeHeaders, ", "))
			}
			if isPreflight {
				if len(opts.AllowMethods) > 0 {
					ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(opts.AllowMethods, ", "))
				}
				if len(opts.AllowHeaders) > 0 {
					ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(opts.AllowHeaders, ", "))
				}
				if opts.MaxAge > 0 {
					ctx.SetHeader("Access-Control-Max-Age", strconv.Itoa(opts.MaxAge))
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
func BasicAuth(authFunc func(name string, password string) (ok bool, err error)) Handle {
	return BasicAuthWithRealm("", authFunc)
}

// BasicAuthWithRealm returns a Basic HTTP Authorization middleware with realm.
func BasicAuthWithRealm(realm string, authFunc func(name string, password string) (ok bool, err error)) Handle {
	return func(ctx *Context) {
		if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
			if authType, authData := utils.SplitByFirstByte(auth, ' '); len(authData) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(authData)
				if e != nil {
					return
				}

				name, password := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err := authFunc(name, password)
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

// SIDStore returns a SIDStore middleware.
func SIDStore(sidStore session.SIDStore) Handle {
	return func(ctx *Context) {
		if sidStore != nil {
			ctx.sidStore = sidStore
		}
		ctx.Next()
	}
}

// SessionPool returns a SessionPool middleware.
func SessionPool(pool session.Pool) Handle {
	return func(ctx *Context) {
		if pool != nil {
			ctx.sessionPool = pool
		}
		ctx.Next()
	}
}

// Static returns a file static serve middleware.
func Static(root string, fallbackPath ...string) Handle {
	return func(ctx *Context) {
		var fallback bool
		var filepath string
		if val := ctx.URL.Param("filepath"); val != "" {
			filepath = val
		} else if val := ctx.URL.Param("path"); val != "" {
			filepath = val
		} else {
			filepath = ctx.URL.RoutePath
		}
		fp := path.Join(root, utils.CleanPath(filepath))
	Re:
		fi, err := os.Stat(fp)
		if err != nil {
			if os.IsExist(err) {
				ctx.Error(err.Error(), 500)
				return
			}

			if fl := len(fallbackPath); fl > 0 && !fallback {
				fp = path.Join(root, utils.CleanPath(fallbackPath[0]))
				fallback = true
				goto Re
			}

			ctx.End(404)
			return
		}

		if fi.IsDir() {
			fp = path.Join(fp, "index.html")
			goto Re
		}

		ctx.File(fp)
	}
}
