package rex

import (
	"net/http"

	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
)

type AutoTLSConfig struct {
	Enable   bool     `json:"enable"`
	CacheDir string   `json:"cacheDir"`
	CacheURL string   `json:"cacheUrl"`
	Hosts    []string `json:"hosts"`
}

type HTTPSConfig struct {
	Port     uint16        `json:"port"`
	CertFile string        `json:"certFile"`
	KeyFile  string        `json:"keyFile"`
	AutoTLS  AutoTLSConfig `json:"autotls"`
}

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
