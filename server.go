package rex

import (
	"fmt"
	"net/http"
	"os"
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
	for _, apis := range globalAPIServices {
		mux.RegisterAPIService(apis)
	}

	var wg sync.WaitGroup

	if config.Port > 0 {
		wg.Add(1)
		go func() {
			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), config.Port),
				Handler:        mux,
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			err := serv.ListenAndServe()
			if err != nil {
				fmt.Println("rex server shutdown:", err)
			}
			serv.Shutdown(nil)
			wg.Done()
		}()
	}

	if https := config.HTTPS; https.Port > 0 {
		wg.Add(1)
		go func() {
			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), https.Port),
				Handler:        mux,
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			if https.Autocert.Enable {
				m := &autocert.Manager{
					Prompt: autocert.AcceptTOS,
				}
				m.Cache, _ = cache.New("memory")
				if https.Autocert.CacheDir != "" {
					fi, err := os.Stat(https.Autocert.CacheDir)
					if err == nil && fi.IsDir() {
						m.Cache = autocert.DirCache(https.Autocert.CacheDir)
					}
				}
				if len(https.Autocert.HostWhitelist) > 0 {
					m.HostPolicy = autocert.HostWhitelist(https.Autocert.HostWhitelist...)
				}
				serv.TLSConfig = m.TLSConfig()
			}
			err := serv.ListenAndServeTLS(https.CertFile, https.KeyFile)
			if err != nil {
				fmt.Println("rex server(https) shutdown:", err)
			}
			serv.Shutdown(nil)
			wg.Done()
		}()
	}

	wg.Wait()
}
