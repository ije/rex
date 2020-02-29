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

var gRESTs = map[string][][]*REST{}

// Serve serves a rex server.
func Serve(config Config) {
	if config.Logger == nil {
		config.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	_gRESTs := linkRESTs()
	for _, prefixs := range _gRESTs {
		for _, rests := range prefixs {
			for _, rest := range rests {
				if rest.AccessLogger == nil && config.AccessLogger != nil {
					rest.AccessLogger = config.AccessLogger
				}
				if rest.Logger == nil {
					rest.Logger = config.Logger
				}
				rest.sendError = config.Debug
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
				Handler:        &mux{_gRESTs, config.TLS.AutoRedirect},
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
				Handler:        &mux{_gRESTs, config.TLS.AutoRedirect},
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

// Start starts a REX server.
func Start(port uint16) {
	Serve(Config{
		Port: port,
	})
}

// StartTLS starts a REX server with TLS.
func StartTLS(port uint16, certFile string, keyFile string) {
	Serve(Config{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	})
}

// StartAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
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

func applyREST(rest *REST) {
	// clean up
	for host, prefixs := range gRESTs {
		var _prefixs [][]*REST
		for _, rests := range prefixs {
			var _rests []*REST
			for _, _rest := range rests {
				if _rest != rest {
					_rests = append(_rests, _rest)
				}
			}
			if len(_rests) > 0 {
				_prefixs = append(_prefixs, _rests)
			}
		}
		if len(_prefixs) > 0 {
			gRESTs[host] = _prefixs
		}
	}

	// append or insert
	prefixs, ok := gRESTs[rest.host]
	if ok {
		for i, rests := range prefixs {
			if rest.prefix == rests[0].prefix {
				prefixs[i] = append(rests, rest)
				return
			}
		}
	}
	if len(prefixs) == 0 {
		prefixs = [][]*REST{[]*REST{rest}}
	} else {
		insertIndex := 0
		for i, rests := range prefixs {
			if len(rest.prefix) > len(rests[0].prefix) {
				insertIndex = i
				break
			}
		}
		tmp := make([][]*REST, len(prefixs)+1)
		copy(tmp, prefixs[:insertIndex])
		copy(tmp[insertIndex+1:], prefixs[insertIndex:])
		tmp[insertIndex] = []*REST{rest}
		prefixs = tmp
	}
	gRESTs[rest.host] = prefixs
}

func linkRESTs() map[string][][]*REST {
	_gRESTs := map[string][][]*REST{}
	for host, prefixs := range gRESTs {
		var _prefixs [][]*REST
		for _, rests := range prefixs {
			var _rests []*REST
			for _, rest := range rests {
				if rest.router != nil {
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

	return _gRESTs
}
