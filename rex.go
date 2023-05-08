// Package rex provides a simple & light-weight REST server in golang
package rex

import (
	"fmt"
	"net/http"
	"os"
	"time"

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

// Serve serves a rex server.
func Serve(config ServerConfig) chan error {
	c := make(chan error, 1)

	go serve(&config, c)

	if tls := config.TLS; tls.AutoTLS.AcceptTOS || (tls.CertFile != "" && tls.KeyFile != "") {
		go serveTLS(&config, c)
	}

	return c
}

func serve(config *ServerConfig, c chan error) {
	port := config.Port
	if port == 0 {
		port = 80
	}
	serv := &http.Server{
		Addr:           fmt.Sprintf(("%s:%d"), config.Host, port),
		Handler:        &mux{config.TLS.AutoRedirect},
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: int(config.MaxHeaderBytes),
	}
	err := serv.ListenAndServe()
	if err != nil {
		c <- fmt.Errorf("rex server shutdown: %v", err)
	}
}

func serveTLS(config *ServerConfig, c chan error) {
	tls := config.TLS
	port := tls.Port
	if port == 0 {
		port = 443
	}
	serv := &http.Server{
		Addr:           fmt.Sprintf(("%s:%d"), config.Host, port),
		Handler:        &mux{},
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: int(config.MaxHeaderBytes),
	}
	if tls.AutoTLS.AcceptTOS {
		m := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
		}
		if tls.AutoTLS.Cache != nil {
			m.Cache = tls.AutoTLS.Cache
		} else if cacheDir := tls.AutoTLS.CacheDir; cacheDir != "" {
			fi, err := os.Stat(cacheDir)
			if err == nil && !fi.IsDir() {
				c <- fmt.Errorf("AutoTLS: invalid cache dir '%s'", cacheDir)
				return
			}
			if err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(cacheDir, 0755)
				if err != nil {
					c <- fmt.Errorf("[error] AutoTLS: can't create the cache dir '%s'", cacheDir)
					return
				}
			}
			m.Cache = autocert.DirCache(cacheDir)
		}
		if len(tls.AutoTLS.Hosts) > 0 {
			m.HostPolicy = autocert.HostWhitelist(tls.AutoTLS.Hosts...)
		}
		serv.TLSConfig = m.TLSConfig()
	}
	err := serv.ListenAndServeTLS(tls.CertFile, tls.KeyFile)
	if err != nil {
		c <- fmt.Errorf("rex server(https) shutdown: %v", err)
	}
}

// Start starts a REX server.
func Start(port uint16) chan error {
	return Serve(ServerConfig{
		Port: port,
	})
}

// StartWithTLS starts a REX server with TLS.
func StartWithTLS(port uint16, certFile string, keyFile string) chan error {
	return Serve(ServerConfig{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	})
}

// StartWithAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
func StartWithAutoTLS(port uint16, hosts ...string) chan error {
	return Serve(ServerConfig{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				Hosts:     hosts,
				CacheDir:  "/var/rex/autotls",
			},
		},
	})
}
