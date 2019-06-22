package rex

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// Serve serves the rex server
func Serve(config Config) {
	if config.Port == 0 {
		config.Port = 80
	}

	if config.Logger == nil {
		config.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	for _, rest := range gRESTs {
		config.Logger.Println(rest.prefix)
		rest.SendError = config.Debug
		if rest.AccessLogger == nil {
			rest.AccessLogger = config.AccessLogger
		}
		if rest.Logger == nil {
			rest.Logger = config.Logger
		}
	}

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wh := w.Header()
		wh.Set("Connection", "keep-alive")
		wh.Set("Server", "rex-serv")

		if len(gRESTs) > 0 {
			for _, rest := range gRESTs {
				if rest.prefix != "" && strings.HasPrefix(r.URL.Path, "/"+rest.prefix) {
					rest.ServeHTTP(w, r)
					return
				}
			}
			if rest := gRESTs[len(gRESTs)-1]; rest.prefix == "" {
				rest.ServeHTTP(w, r)
				return
			}
		}

		http.NotFound(w, r)
	})

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
			config.Logger.Println("[error] rex server shutdown:", err)
		}
		serv.Shutdown(nil)
	}()

	if https := config.HTTPS; (https.CertFile != "" && https.KeyFile != "") || (https.AutoTLS.Enable && (https.AutoTLS.CacheDir != "" || https.AutoTLS.Cache != nil)) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			port := https.Port
			if port == 0 {
				port = 443
			}
			serv := &http.Server{
				Addr:           fmt.Sprintf((":%d"), port),
				Handler:        mux,
				ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
				WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
				MaxHeaderBytes: int(config.MaxHeaderBytes),
			}
			if https.AutoTLS.Enable {
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
				serv.TLSConfig = m.TLSConfig()
			}
			err := serv.ListenAndServeTLS(https.CertFile, https.KeyFile)
			if err != nil {
				config.Logger.Println("[error] rex server(https) shutdown:", err)
			}
			serv.Shutdown(nil)
		}()
	}

	if config.Debug {
		config.Logger.Println("[debug] rex server startd.")
	}

	wg.Wait()
}
