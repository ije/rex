package rex

import (
	"net/http"

	"github.com/ije/gox/log"
)

type Config struct {
	Port              uint16            `json:"port"`
	Root              string            `json:"root"`
	ServerName        string            `json:"serverName"`
	CustomHTTPHeaders map[string]string `json:"customHTTPHeaders"`
	SessionCookieName string            `json:"sessionCookieName"`
	HostRedirectRule  string            `json:"hostRedirectRule"`
	ReadTimeout       uint32            `json:"readTimeout"`
	WriteTimeout      uint32            `json:"writeTimeout"`
	MaxHeaderBytes    uint32            `json:"maxHeaderBytes"`
	Debug             bool              `json:"debug"`
	NotFoundHandler   http.Handler      `json:"-"`
	ErrorLogger       *log.Logger       `json:"-"`
	AccessLogger      *log.Logger       `json:"-"`
}
