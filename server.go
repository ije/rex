package rex

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/ije/gox/cache"
	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
	"golang.org/x/crypto/acme/autocert"
)

// Serve serves the rex server
func Serve(config Config) {
	if config.Port == 0 {
		config.Port = 80
	}
	if config.Logger == nil {
		config.Logger = &log.Logger{}
	}
	if config.SessionManager == nil {
		config.SessionManager = session.NewMemorySessionManager(time.Hour / 2)
	}
	if !config.Debug {
		config.Logger.SetLevelByName("info")
		config.Logger.SetQuite(true)
		if config.AccessLogger != nil {
			config.AccessLogger.SetQuite(true)
		}
	}

	mux := &Mux{Config: config}
	for _, rest := range gRESTs {
		mux.RegisterREST(rest)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		serv := &http.Server{
			Addr:           fmt.Sprintf((":%d"), config.Port),
			Handler:        mux,
			ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
			WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
			MaxHeaderBytes: int(config.MaxHeaderBytes),
		}
		err := serv.ListenAndServe()
		if err != nil {
			config.Logger.Error("rex server shutdown:", err)
		}
		serv.Shutdown(nil)
	}()

	if https := config.HTTPS; https.Port > 0 && https.Port != config.Port {
		wg.Add(1)
		go func() {
			defer wg.Done()
			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), https.Port),
				Handler:        mux,
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			if https.AutoTLS.Enable && !config.Debug {
				m := &autocert.Manager{
					Prompt: autocert.AcceptTOS,
				}
				if https.AutoTLS.CacheDir != "" {
					fi, err := os.Stat(https.AutoTLS.CacheDir)
					if err == nil && !fi.IsDir() {
						config.Logger.Errorf("can not init tls: bad cache dir '%s'", https.AutoTLS.CacheDir)
						return
					}
					m.Cache = autocert.DirCache(https.AutoTLS.CacheDir)
				} else if https.AutoTLS.CacheURL != "" {
					cache, err := cache.New(https.AutoTLS.CacheURL)
					if err != nil {
						config.Logger.Error("can not init tls:", err)
						return
					}
					m.Cache = cache
				} else {
					m.Cache = autocert.DirCache(path.Join(os.TempDir(), ".rex-cert-cache"))
				}
				if len(https.AutoTLS.Hosts) > 0 {
					m.HostPolicy = autocert.HostWhitelist(https.AutoTLS.Hosts...)
				}
				serv.TLSConfig = m.TLSConfig()
			}
			err := serv.ListenAndServeTLS(https.CertFile, https.KeyFile)
			if err != nil {
				config.Logger.Error("rex server(https) shutdown:", err)
			}
			serv.Shutdown(nil)
		}()
	}

	wg.Wait()
}
