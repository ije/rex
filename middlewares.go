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

func CORS(cors CORSOptions) RESTHandle {
	return func(ctx *Context) {
		if len(cors.AllowOrigin) > 0 {
			ctx.SetHeader("Access-Control-Allow-Origin", cors.AllowOrigin)
			if cors.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if ctx.R.Method == "OPTIONS" {
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

func ACLAuth(getUserFunc func(ctx *Context) acl.User) RESTHandle {
	return func(ctx *Context) {
		if getUserFunc != nil {
			ctx.aclUser = getUserFunc(ctx)
		}
		ctx.Next()
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware.
func BasicAuth(realm string, check func(name string, password string) (ok bool, err error)) RESTHandle {
	return func(ctx *Context) {
		if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
			if authType, combination := utils.SplitByFirstByte(auth, ' '); len(combination) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(combination)
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

		ctx.SetHeader("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
		ctx.W.WriteHeader(401)
	}
}

func Session(manager SessionManager) RESTHandle {
	return func(ctx *Context) {
		pool := manager.Pool
		sidStore := manager.SIDStore
		if pool == nil {
			pool = defaultSessionManager.Pool
		}
		if sidStore == nil {
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
