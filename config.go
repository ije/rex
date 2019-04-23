package rex

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
)

type Config struct {
	Debug             bool              `json:"debug"`
	Port              uint16            `json:"port"`
	HTTPS             HTTPSConfig       `json:"https"`
	Root              string            `json:"root"`
	ServerName        string            `json:"serverName"`
	CustomHTTPHeaders map[string]string `json:"customHTTPHeaders"`
	SessionCookieName string            `json:"sessionCookieName"`
	HostRedirectRule  string            `json:"hostRedirectRule"`
	ReadTimeout       uint32            `json:"readTimeout"`
	WriteTimeout      uint32            `json:"writeTimeout"`
	MaxHeaderBytes    uint32            `json:"maxHeaderBytes"`
	NotFoundHandler   http.Handler      `json:"-"`
	SessionManager    session.Manager   `json:"-"`
	Logger            *log.Logger       `json:"-"`
	AccessLogger      *log.Logger       `json:"-"`
}

type HTTPSConfig struct {
	Port     uint16        `json:"port"`
	CertFile string        `json:"certFile"`
	KeyFile  string        `json:"keyFile"`
	AutoTLS  AutoTLSConfig `json:"autotls"`
}

type AutoTLSConfig struct {
	Enable   bool     `json:"enable"`
	CacheDir string   `json:"cacheDir"`
	CacheURL string   `json:"cacheUrl"`
	Hosts    []string `json:"hosts"`
}

type CORSConfig struct {
	AllowOrigin      string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

func PublicCORS() CORSConfig {
	return CORSConfig{
		AllowOrigin:      "*",
		AllowMethods:     []string{"OPTIONS", "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Accept", "Accept-Encoding", "Accept-Lang", "Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           60,
	}
}

func (cors CORSConfig) Apply(w http.ResponseWriter) {
	if len(cors.AllowOrigin) > 0 {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", cors.AllowOrigin)
		h.Set("Vary", "Origin")
		if len(cors.AllowMethods) > 0 {
			h.Set("Access-Control-Allow-Methods", strings.Join(cors.AllowMethods, ", "))
		}
		if len(cors.AllowHeaders) > 0 {
			h.Set("Access-Control-Allow-Headers", strings.Join(cors.AllowHeaders, ", "))
		}
		if cors.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		if cors.MaxAge > 0 {
			h.Set("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
		}
	}
}
