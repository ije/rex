// Package rex provides a simple & light-weight REST server in golang
package rex

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// Serve serves a rex server.
func Serve(config ServerConfig) {
	var wg sync.WaitGroup

	if config.Port > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), config.Port),
				Handler:        &mux{config.TLS.AutoRedirect},
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			err := serv.ListenAndServe()
			if err != nil {
				log.Println("[error] rex server shutdown:", err)
			}
		}()
	}

	if https := config.TLS; (https.CertFile != "" && https.KeyFile != "") || https.AutoTLS.AcceptTOS {
		port := https.Port
		if port == 0 {
			port = 443
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			servs := &http.Server{
				Addr:           fmt.Sprintf((":%d"), port),
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
				} else if https.AutoTLS.CacheDir != "" {
					fi, err := os.Stat(https.AutoTLS.CacheDir)
					if err == nil && !fi.IsDir() {
						log.Printf("[error] AutoTLS: invalid cache dir '%s'", https.AutoTLS.CacheDir)
						return
					}
					if err != nil && os.IsNotExist(err) {
						err = os.MkdirAll(https.AutoTLS.CacheDir, 0755)
						if err != nil {
							log.Printf("[error] AutoTLS: can't create the cache dir '%s'", https.AutoTLS.CacheDir)
							return
						}
					}
					m.Cache = autocert.DirCache(https.AutoTLS.CacheDir)
				}
				if len(https.AutoTLS.Hosts) > 0 {
					m.HostPolicy = autocert.HostWhitelist(https.AutoTLS.Hosts...)
				}
				servs.TLSConfig = m.TLSConfig()
			}
			err := servs.ListenAndServeTLS(https.CertFile, https.KeyFile)
			if err != nil {
				log.Println("[error] rex server(https) shutdown:", err)
			}
		}()
	}

	log.Println("[info] rex server started.")
	wg.Wait()
}

// Start starts a REX server.
func Start(port uint16) {
	Serve(ServerConfig{
		Port: port,
	})
}

// StartTLS starts a REX server with TLS.
func StartTLS(port uint16, certFile string, keyFile string) {
	Serve(ServerConfig{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	})
}

// StartAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
func StartAutoTLS(port uint16, hosts ...string) {
	Serve(ServerConfig{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				Hosts:     hosts,
				CacheDir:  "./.certs",
			},
		},
	})
}
