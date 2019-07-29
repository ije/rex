package rex

import (
	"golang.org/x/crypto/acme/autocert"
)

// Config contains the options to run REX server.
type Config struct {
	Port           uint16      `json:"port"`
	HTTPS          HTTPSConfig `json:"https"`
	ReadTimeout    uint32      `json:"readTimeout"`
	WriteTimeout   uint32      `json:"writeTimeout"`
	MaxHeaderBytes uint32      `json:"maxHeaderBytes"`
	Debug          bool        `json:"debug"`
	Logger         Logger      `json:"-"`
	AccessLogger   Logger      `json:"-"`
}

// HTTPSConfig contains the options to support https.
type HTTPSConfig struct {
	Port     uint16        `json:"port"`
	CertFile string        `json:"certFile"`
	KeyFile  string        `json:"keyFile"`
	AutoTLS  AutoTLSConfig `json:"autotls"`
}

// AutoTLSConfig contains the options to support autocert by Let's Encrypto SSL.
type AutoTLSConfig struct {
	Enable   bool           `json:"enable"`
	Hosts    []string       `json:"hosts"`
	CacheDir string         `json:"cacheDir"`
	Cache    autocert.Cache `json:"-"`
}

// CORSOptions contains the options to CORS.
type CORSOptions struct {
	AllowOrigin      string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

// Logger is a Logger contains Println and Printf methods
type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}
