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

// Serve serves the rex server
func Serve(config Config) {
	if config.Logger == nil {
		config.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	_gRESTs := map[string][][]*REST{}
	for host, prefixs := range gRESTs {
		var _prefixs [][]*REST
		for _, rests := range prefixs {
			var _rests []*REST
			for _, rest := range rests {
				if rest.router != nil {
					if rest.AccessLogger == nil && config.AccessLogger != nil {
						rest.AccessLogger = config.AccessLogger
					}
					if rest.Logger == nil {
						rest.Logger = config.Logger
					}
					_rests = append(_rests, rest)
				}
			}
			if len(_rests) > 0 {
				_prefixs = append(_prefixs, _rests)
			}
		}
		if len(_prefixs) > 0 {
			_gRESTs[host] = _prefixs
		}
	}

	for _, prefixs := range _gRESTs {
		for _, rests := range prefixs {
			if len(rests) > 1 {
				for index, rest := range rests {
					func(index int, rest *REST, rests []*REST) {
						rest.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
							if index+1 <= len(rests)-1 {
								rests[index+1].ServeHTTP(w, r)
								return
							}
							if f := rests[0]; f.notFoundHandle != nil {
								f.serve(w, r, nil, f.notFoundHandle)
							} else if rest.notFoundHandle != nil {
								rest.serve(w, r, nil, rest.notFoundHandle)
							} else {
								rest.serve(w, r, nil, func(ctx *Context) {
									ctx.End(404)
								})
							}
						})
					}(index, rest, rests)
				}
			}
		}
	}

	var wg sync.WaitGroup

	if config.Port > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), config.Port),
				Handler:        &mux{rests: _gRESTs, config: config},
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			err := serv.ListenAndServe()
			if err != nil {
				config.Logger.Println("[error] rex server shutdown:", err)
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
				Handler:        &mux{rests: _gRESTs, config: config},
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
						config.Logger.Printf("[error] AutoTLS: invalid cache dir '%s'", https.AutoTLS.CacheDir)
						return
					}
					if err != nil && os.IsNotExist(err) {
						err = os.MkdirAll(https.AutoTLS.CacheDir, 0755)
						if err != nil {
							config.Logger.Printf("[error] AutoTLS: can't create the cache dir '%s'", https.AutoTLS.CacheDir)
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
				config.Logger.Println("[error] rex server(https) shutdown:", err)
			}
		}()
	}

	config.Logger.Println("[info] rex server started.")
	wg.Wait()
}

// Start starts an HTTP server.
func Start(port uint16) {
	Serve(Config{
		Port: port,
	})
}

// StartTLS starts an HTTPS server.
func StartTLS(port uint16, certFile string, keyFile string) {
	Serve(Config{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	})
}

// StartAutoTLS starts an HTTPS server using autocert with Let's Encrypto SSL
func StartAutoTLS(port uint16, hosts ...string) {
	Serve(Config{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				Hosts:     hosts,
				CacheDir:  "./.rex-cert-cache",
			},
		},
	})
}
