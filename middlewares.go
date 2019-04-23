package rex

import (
	"encoding/base64"
	"fmt"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/acl"
)

func CORS(cors CORSConfig) RESTHandle {
	return func(ctx *Context) {
		cors.Apply(ctx.W)
		ctx.Next()
	}
}

func SetPrivileges(privileges ...string) RESTHandle {
	return func(ctx *Context) {
		for _, p := range privileges {
			ctx.privileges[p] = struct{}{}
		}
		ctx.Next()
	}
}

func BasicAuth(realm string, check func(user string, pass string) (ok bool, err error)) RESTHandle {
	return func(ctx *Context) {
		if auth := ctx.R.Header.Get("Authorization"); len(auth) > 0 {
			if authType, combination := utils.SplitByFirstByte(auth, ' '); len(combination) > 0 && authType == "Basic" {
				authInfo, e := base64.StdEncoding.DecodeString(combination)
				if e != nil {
					return
				}

				user, pass := utils.SplitByFirstByte(string(authInfo), ':')
				ok, err := check(user, pass)
				if err != nil {
					ctx.Error(err)
					return
				} else if ok {
					ctx.basicUser = acl.BasicUser{
						Name: user,
						Pass: pass,
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
