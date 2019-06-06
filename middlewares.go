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
	"github.com/ije/rex/acl"
)

func Header(key string, value string) RESTHandle {
	return func(ctx *Context) {
		if key != "" {
			ctx.SetHeader(key, value)
		}
		ctx.Next()
	}
}

func HTTPS() RESTHandle {
	return func(ctx *Context) {
		if ctx.R.TLS == nil {
			code := 301
			if ctx.R.Method != "GET" {
				code = 307
			}
			ctx.Redirect(code, fmt.Sprintf("https://%s/%s", ctx.R.Host, ctx.R.RequestURI))
			return
		}
		ctx.Next()
	}
}

func CORS(opts CORSOptions) RESTHandle {
	return func(ctx *Context) {
		if len(opts.AllowOrigin) > 0 {
			ctx.SetHeader("Access-Control-Allow-Origin", opts.AllowOrigin)
			if opts.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if ctx.R.Method == "OPTIONS" {
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
		}
		ctx.Next()
	}
}

func Privileges(privileges ...string) RESTHandle {
	return func(ctx *Context) {
		for _, p := range privileges {
			if p != "" {
				ctx.privileges[p] = struct{}{}
			}
		}
		ctx.Next()
	}
}

// ACLAuth returns a ACL Authorization middleware.
func ACLAuth(getUserFunc func(ctx *Context) (acl.User, error)) RESTHandle {
	return func(ctx *Context) {
		if getUserFunc != nil {
			var err error
			ctx.aclUser, err = getUserFunc(&Context{
				W:              ctx.W,
				R:              ctx.R,
				URL:            ctx.URL,
				State:          ctx.State,
				handles:        []RESTHandle{},
				handleIndex:    -1,
				privileges:     ctx.privileges,
				sessionManager: ctx.sessionManager,
				rest:           ctx.rest,
			})
			if err != nil {
				ctx.Error(err)
				return
			}
		}
		ctx.Next()
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware.
func BasicAuth(realm string, check func(name string, password string) (ok bool, err error)) RESTHandle {
	return func(ctx *Context) {
		if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
			if authType, authSecret := utils.SplitByFirstByte(auth, ' '); len(authSecret) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(authSecret)
				if e != nil {
					return
				}

				name, password := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err := check(name, password)
				if err != nil {
					ctx.Error(err)
					return
				} else if ok {
					ctx.basicUser = acl.BasicUser{
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

func Session(manager SessionManager) RESTHandle {
	return func(ctx *Context) {
		pool := manager.Pool
		sidStore := manager.SIDStore
		if pool == nil && sidStore == nil {
			ctx.Next()
			return
		}

		if pool == nil {
			pool = defaultSessionManager.Pool
		} else if sidStore == nil {
			sidStore = defaultSessionManager.SIDStore
		}
		ctx.sessionManager = &SessionManager{
			SIDStore: sidStore,
			Pool:     pool,
		}
		ctx.Next()
	}
}

func Static(root string, fallbackPaths ...string) RESTHandle {
	return func(ctx *Context) {
		fp := path.Join(root, utils.CleanPath(ctx.URL.Path))
		fallbackIndex := 0
	Re:
		fi, err := os.Stat(fp)
		if err != nil {
			if os.IsExist(err) {
				ctx.Error(err)
				return
			}

			if fl := len(fallbackPaths); fl > 0 && fallbackIndex < fl {
				if fallbackPaths[fallbackIndex] != "" {
					fp = path.Join(root, utils.CleanPath(fallbackPaths[fallbackIndex]))
				}
				fallbackIndex++
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
