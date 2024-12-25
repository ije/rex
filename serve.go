// Package rex provides a simple & light-weight REST server in golang
package rex

import (
	"context"
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

// serve starts a REX server.
func serve(ctx context.Context, config *ServerConfig, c chan error) {
	port := config.Port
	if port == 0 {
		port = 80
	}
	serv := &http.Server{
		Addr:           fmt.Sprintf(("%s:%d"), config.Host, port),
		Handler:        defaultMux,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: int(config.MaxHeaderBytes),
	}
	if ctx != nil {
		go func() {
			<-ctx.Done()
			serv.Close()
		}()
	}
	c <- serv.ListenAndServe()
}

// serveTLS starts a REX server with TLS.
func serveTLS(ctx context.Context, config *ServerConfig, c chan error) {
	tls := config.TLS
	port := tls.Port
	if port == 0 {
		port = 443
	}
	serv := &http.Server{
		Addr:           fmt.Sprintf(("%s:%d"), config.Host, port),
		Handler:        defaultMux,
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
					c <- fmt.Errorf("AutoTLS: can't create the cache dir '%s'", cacheDir)
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
	if ctx != nil {
		go func() {
			<-ctx.Done()
			serv.Close()
		}()
	}
	c <- serv.ListenAndServeTLS(tls.CertFile, tls.KeyFile)
}

// Serve serves a REX server.
func Serve(config ServerConfig) chan error {
	c := make(chan error, 1)

	if tls := config.TLS; tls.AutoTLS.AcceptTOS || (tls.CertFile != "" && tls.KeyFile != "") {
		c2 := make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		go serve(ctx, &config, c2)
		go serveTLS(ctx, &config, c2)
		err := <-c2
		cancel()
		c <- err
	} else {
		go serve(context.Background(), &config, c)
	}

	return c
}

// Start starts a REX server.
func Start(port uint16) chan error {
	c := make(chan error, 1)
	go serve(context.Background(), &ServerConfig{
		Port: port,
	}, c)
	return c
}

// StartWithTLS starts a REX server with TLS.
func StartWithTLS(port uint16, certFile string, keyFile string) chan error {
	c := make(chan error, 1)
	go serveTLS(context.Background(), &ServerConfig{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	}, c)
	return c
}

// StartWithAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
func StartWithAutoTLS(port uint16, hosts ...string) chan error {
	c := make(chan error, 1)
	go serveTLS(context.Background(), &ServerConfig{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				Hosts:     hosts,
				CacheDir:  "/var/rex/autotls",
			},
		},
	}, c)
	return c
}
