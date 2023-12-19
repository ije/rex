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
			ctx.W.Header().Set(key, value)
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

type SessionOptions struct {
	IdHandler session.IdHandler
	Pool      session.Pool
}

func Session(opts SessionOptions) Handle {
	return func(ctx *Context) interface{} {
		if opts.IdHandler != nil {
			ctx.sessionIdHandler = opts.IdHandler
		}
		if opts.Pool != nil {
			ctx.sessionPool = opts.Pool
		}
		return nil
	}
}

// CORS is a configuration container to setup the CORS middleware.
type CORS struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string
	// AllowOriginFunc is a custom function to validate the origin. It take the
	// origin as argument and returns true if allowed or false otherwise. If
	// this option is set, the content of `AllowedOrigins` is ignored.
	AllowOriginFunc func(origin string) bool
	// AllowOriginRequestFunc is a custom function to validate the origin. It
	// takes the HTTP Request object and the origin as argument and returns true
	// if allowed or false otherwise. If headers are used take the decision,
	// consider using AllowOriginVaryRequestFunc instead. If this option is set,
	// the content of `AllowedOrigins`, `AllowOriginFunc` are ignored.
	AllowOriginRequestFunc func(r *http.Request, origin string) bool
	// AllowOriginVaryRequestFunc is a custom function to validate the origin.
	// It takes the HTTP Request object and the origin as argument and returns
	// true if allowed or false otherwise with a list of headers used to take
	// that decision if any so they can be added to the Vary header. If this
	// option is set, the content of `AllowedOrigins`, `AllowOriginFunc` and
	// `AllowOriginRequestFunc` are ignored.
	AllowOriginVaryRequestFunc func(r *http.Request, origin string) (bool, []string)
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [].
	AllowedHeaders []string
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Default value is 0, which stands for no
	// Access-Control-Max-Age header to be sent back, resulting in browsers
	// using their default value (5s by spec). If you need to force a 0 max-age,
	// set `MaxAge` to a negative value (ie: -1).
	MaxAge int
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// AllowPrivateNetwork indicates whether to accept cross-origin requests over a
	// private network.
	AllowPrivateNetwork bool
	// OptionsPassthrough instructs preflight to let other potential next handlers to
	// process the OPTIONS method. Turn this on if your application handles OPTIONS.
	OptionsPassthrough bool
	// Provides a status code to use for successful OPTIONS requests.
	// Default value is http.StatusNoContent (204).
	OptionsSuccessStatus int
	// Debugging flag adds additional output to debug server side CORS issues
	Debug bool
	// Adds a custom logger, implies Debug is true
	Logger Logger
}

// CorsAllowAll create a new Cors handler with permissive configuration allowing all
// origins with all standard methods with any header and credentials.
func CorsAllowAll() CORS {
	return CORS{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}
}

// Cors returns a Cors middleware to handle CORS.
func Cors(c CORS) Handle {
	cors := cors.New(cors.Options{
		AllowedOrigins:             c.AllowedOrigins,
		AllowOriginFunc:            c.AllowOriginFunc,
		AllowOriginRequestFunc:     c.AllowOriginRequestFunc,
		AllowOriginVaryRequestFunc: c.AllowOriginVaryRequestFunc,
		AllowedMethods:             c.AllowedMethods,
		AllowedHeaders:             c.AllowedHeaders,
		ExposedHeaders:             c.ExposedHeaders,
		MaxAge:                     c.MaxAge,
		AllowCredentials:           c.AllowCredentials,
		AllowPrivateNetwork:        c.AllowPrivateNetwork,
		OptionsPassthrough:         c.OptionsPassthrough,
		OptionsSuccessStatus:       c.OptionsSuccessStatus,
		Debug:                      c.Debug,
		Logger:                     c.Logger,
	})
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
func ACL(permission string) Handle {
	return func(ctx *Context) interface{} {
		if ctx.aclUser != nil {
			permissions := ctx.aclUser.Permissions()
			for _, p := range permissions {
				if p == permission {
					return nil // next
				}
			}
		}
		return &Error{403, "Forbidden"}
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
		ctx.W.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
		return Status(401, "")
	}
}

// ACLAuth returns a ACL authentication middleware.
func ACLAuth(auth func(ctx *Context) ACLUser) Handle {
	return func(ctx *Context) interface{} {
		user := auth(ctx)
		ctx.aclUser = user
		return nil
	}
}

// Compression is REX middleware to enable compress by content type and client `Accept-Encoding`
func Compression() Handle {
	return func(ctx *Context) interface{} {
		ctx.compression = true
		return nil
	}
}

// Static returns a Static middleware handler.
func Static(root, fallback string) Handle {
	return func(ctx *Context) interface{} {
		return FS(root, fallback)
	}
}

// Chain returns a middleware handler that executes handlers in a chain.
func Chain(handles ...Handle) Handle {
	if len(handles) == 0 {
		panic("no handles in the chain")
	}
	return func(ctx *Context) interface{} {
		w, r, path, form, store := ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store
		for _, handle := range handles {
			v := handle(ctx)
			if v != nil {
				return v
			}
			ctx.W, ctx.R, ctx.Path, ctx.Form, ctx.Store = w, r, path, form, store
		}
		return nil
	}
}
