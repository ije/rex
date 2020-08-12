// Package rex provides a simple & light-weight REST server in golang
package rex

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// Serve serves a rex server.
func Serve(config ServerConfig) chan error {
	c := make(chan error, 1)

	if config.Port > 0 {
		go func() {
			serv := &http.Server{
				Addr:           fmt.Sprintf(("%s:%d"), config.Host, config.Port),
				Handler:        &mux{config.TLS.AutoRedirect},
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			err := serv.ListenAndServe()
			if err != nil {
				c <- fmt.Errorf("rex server shutdown: %v", err)
			}
		}()
	}

	if https := config.TLS; https.AutoTLS.AcceptTOS || (https.CertFile != "" && https.KeyFile != "") {
		go func() {
			port := https.Port
			if port == 0 {
				port = 443
			}
			servs := &http.Server{
				Addr:           fmt.Sprintf(("%s:%d"), config.Host, port),
				Handler:        &mux{},
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			if https.AutoTLS.AcceptTOS {
				m := &autocert.Manager{
					Prompt: autocert.AcceptTOS,
				}
				if https.AutoTLS.Cache != nil {
					m.Cache = https.AutoTLS.Cache
				} else if cacheDir := https.AutoTLS.CacheDir; cacheDir != "" {
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
				if len(https.AutoTLS.Hosts) > 0 {
					m.HostPolicy = autocert.HostWhitelist(https.AutoTLS.Hosts...)
				}
				servs.TLSConfig = m.TLSConfig()
			}
			err := servs.ListenAndServeTLS(https.CertFile, https.KeyFile)
			if err != nil {
				c <- fmt.Errorf("rex server(https) shutdown: %v", err)
			}
		}()
	}

	return c
}

// Start starts a REX server.
func Start(port uint16) (err error) {
	err = <-Serve(ServerConfig{
		Port: port,
	})
	return
}

// StartTLS starts a REX server with TLS.
func StartTLS(port uint16, certFile string, keyFile string) (err error) {
	err = <-Serve(ServerConfig{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	})
	return
}

// StartAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
func StartAutoTLS(port uint16, hosts ...string) (err error) {
	err = <-Serve(ServerConfig{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				Hosts:     hosts,
				CacheDir:  "./.certs",
			},
		},
	})
	return
}
