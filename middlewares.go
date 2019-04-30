package rex

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ije/gox/utils"
	"github.com/ije/rex/acl"
	"github.com/ije/rex/session"
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

func ACL(getUser func(ctx *Context) acl.User) RESTHandle {
	return func(ctx *Context) {
		ctx.user = getUser(ctx)
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

func SessionManager(manager session.Manager) RESTHandle {
	return func(ctx *Context) {
		if manager != nil {
			ctx.sessionManager = manager
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
					ctx.basicAuthUser = acl.BasicAuthUser{
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

func Zip(path string) RESTHandle {
	return func(ctx *Context) {
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				ctx.End(404)
			} else {
				ctx.Error(err)
			}
			return
		}
		ctx.SetHeader("Content-Type", "application/zip")
		if fi.IsDir() {
			dir, err := filepath.Abs(path)
			if err != nil {
				ctx.Error(err)
				return
			}

			archive := zip.NewWriter(ctx.W)
			defer archive.Close()

			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}

				header.Name = strings.TrimPrefix(strings.TrimPrefix(path, dir), "/")
				if header.Name == "" {
					return nil
				}

				if info.IsDir() {
					header.Name += "/"
				} else {
					header.Method = zip.Deflate
				}

				gzw, err := archive.CreateHeader(header)
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(gzw, file)
				return err
			})
		} else {
			header, err := zip.FileInfoHeader(fi)
			if err != nil {
				ctx.Error(err)
				return
			}

			file, err := os.Open(path)
			if err != nil {
				ctx.Error(err)
				return
			}
			defer file.Close()

			archive := zip.NewWriter(ctx.W)
			defer archive.Close()

			gzw, err := archive.CreateHeader(header)
			if err != nil {
				ctx.Error(err)
				return
			}

			io.Copy(gzw, file)
		}
	}
}
