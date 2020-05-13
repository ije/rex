package rex

import (
	"io"
	"net/url"

	"github.com/ije/rex/router"
	"golang.org/x/crypto/acme/autocert"
)

// ServerConfig contains options to run the REX server.
type ServerConfig struct {
	Host           string    `json:"host"`
	Port           uint16    `json:"port"`
	TLS            TLSConfig `json:"tls"`
	ReadTimeout    uint32    `json:"readTimeout"`
	WriteTimeout   uint32    `json:"writeTimeout"`
	MaxHeaderBytes uint32    `json:"maxHeaderBytes"`
}

// TLSConfig contains options to support https.
type TLSConfig struct {
	Port         uint16        `json:"port"`
	CertFile     string        `json:"certFile"`
	KeyFile      string        `json:"keyFile"`
	AutoTLS      AutoTLSConfig `json:"autotls"`
	AutoRedirect bool          `json:"autoRedirect"`
}

// AutoTLSConfig contains options to support autocert by Let's Encrypto SSL.
type AutoTLSConfig struct {
	AcceptTOS bool           `json:"acceptTOS"`
	Hosts     []string       `json:"hosts"`
	CacheDir  string         `json:"cacheDir"`
	Cache     autocert.Cache `json:"-"`
}

// CORS contains options to CORS.
type CORS struct {
	AllowAllOrigins  bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

// Handle defines the REST handle
type Handle func(ctx *Context)

// A ACLUser interface contains the Permissions method that returns the permission IDs
type ACLUser interface {
	Permissions() []string
}

// A Template interface contains the Execute method.
type Template interface {
	Execute(wr io.Writer, data interface{}) error
}

// A Logger interface contains the Printf method.
type Logger interface {
	Printf(format string, v ...interface{})
}

// A URL is a url.URL extends router.Params and RoutePath
type URL struct {
	Params    router.Params
	RoutePath string
	*url.URL
}

// Param returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (url *URL) Param(name string) string {
	if url.Params != nil {
		return url.Params.ByName(name)
	}
	return ""
}

// recoverMessage is an Error to pass recover message when panic happen.
type recoverMessage struct {
	code    int
	message string
}

func (err *recoverMessage) Error() string {
	return err.message
}
