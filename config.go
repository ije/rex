package rex

import (
	"net/http"

	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
)

type AutocertConfig struct {
	HostWhitelist []string `json:"hostWhitelist"`
	CacheDir      string   `json:"cacheDir"`
}

type HTTPSConfig struct {
	Port     uint16         `json:"port"`
	Autocert AutocertConfig `json:"autocert"`
}

type Config struct {
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
	Debug             bool              `json:"debug"`
	SessionManager    session.Manager   `json:"-"`
	NotFoundHandler   http.Handler      `json:"-"`
	Logger            *log.Logger       `json:"-"`
	AccessLogger      *log.Logger       `json:"-"`
}
