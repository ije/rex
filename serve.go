package rex

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ije/gox/utils"
	"golang.org/x/crypto/acme/autocert"
)

var gRESTs = map[string][]*REST{}

// Serve serves the rex server
func Serve(config Config) {
	if config.Logger == nil {
		config.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	for _, prefixs := range gRESTs {
		for _, rest := range prefixs {
			if rest.AccessLogger == nil && config.AccessLogger != nil {
				rest.AccessLogger = config.AccessLogger
			}
			if rest.Logger == nil {
				rest.Logger = config.Logger
			}
		}
	}

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wh := w.Header()
		wh.Set("Connection", "keep-alive")
		wh.Set("Server", "rex-serv")

		if config.TLS.AutoRedirect && r.TLS == nil {
			code := 301
			if r.Method != "GET" {
				code = 307
			}
			http.Redirect(w, r, fmt.Sprintf("https://%s/%s", r.Host, r.RequestURI), code)
			return
		}

		host, _ := utils.SplitByLastByte(r.Host, ':')
		prefixs, ok := gRESTs[host]
		if !ok && strings.HasPrefix(host, "www.") {
			prefixs, ok = gRESTs[strings.TrimPrefix(host, "www.")]
		}
		if !ok {
			prefixs, ok = gRESTs["*"]
		}
		if !ok {
			http.NotFound(w, r)
			return
		}

		if len(prefixs) > 0 {
			for _, rest := range prefixs {
				if rest.prefix != "" && strings.HasPrefix(r.URL.Path, "/"+rest.prefix) {
					rest.ServeHTTP(w, r)
					return
				}
			}
			if rest := prefixs[len(prefixs)-1]; rest.prefix == "" {
				rest.ServeHTTP(w, r)
				return
			}
		}

		http.NotFound(w, r)
	})

	var wg sync.WaitGroup

	if config.Port > 0 {
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
				Handler:        mux,
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
			servs.Shutdown(nil)
		}()
	}

	config.Logger.Println("rex server started.")
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
			},
		},
	})
}

func gREST(host string, prefix string) *REST {
	prefixs, ok := gRESTs[host]
	if ok {
		for _, rest := range prefixs {
			if rest.prefix == prefix {
				return rest
			}
		}
	}

	rest := &REST{
		host:   host,
		prefix: prefix,
	}
	rest.initRouter()
	if len(prefixs) == 0 {
		prefixs = []*REST{rest}
	} else {
		insertIndex := 0
		for i, r := range prefixs {
			if len(prefix) > len(r.prefix) {
				insertIndex = i
				break
			}
		}
		tmp := make([]*REST, len(prefixs)+1)
		copy(tmp, prefixs[:insertIndex])
		copy(tmp[insertIndex+1:], prefixs[insertIndex:])
		tmp[insertIndex] = rest
		prefixs = tmp
	}
	gRESTs[host] = prefixs

	return rest
}
