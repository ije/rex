// Package rex provides a simple & light-weight REST server in golang
package rex

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// ServerConfig contains options to run the REX server.
type ServerConfig struct {
	Host           string
	Port           uint16
	TLS            TLSConfig
	ReadTimeout    uint32
	WriteTimeout   uint32
	MaxHeaderBytes uint32
}

// TLSConfig contains options to support https.
type TLSConfig struct {
	Port         uint16
	CertFile     string
	KeyFile      string
	AutoTLS      AutoTLSConfig
	AutoRedirect bool
}

// AutoTLSConfig contains options to support autocert by Let's Encrypto SSL.
type AutoTLSConfig struct {
	AcceptTOS bool
	Hosts     []string
	CacheDir  string
	Cache     autocert.Cache
}

// serve starts a REX server.
func serve(ctx context.Context, config *ServerConfig, c chan error) error {
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

	ln, err := net.Listen("tcp", serv.Addr)
	if err != nil {
		close(c)
		return err
	}

	go func() {
		defer ln.Close()
		c <- serv.Serve(ln)
	}()
	return nil
}

// serveTLS starts a REX server with TLS.
func serveTLS(ctx context.Context, config *ServerConfig, c chan error) error {
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
				err = fmt.Errorf("AutoTLS: invalid cache dir '%s'", cacheDir)
				close(c)
				return err
			}
			if err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(cacheDir, 0755)
				if err != nil {
					err = fmt.Errorf("AutoTLS: can't create the cache dir '%s'", cacheDir)
					close(c)
					return err
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

	ln, err := net.Listen("tcp", serv.Addr)
	if err != nil {
		close(c)
		return err
	}

	go func() {
		defer ln.Close()
		c <- serv.ServeTLS(ln, tls.CertFile, tls.KeyFile)
	}()
	return nil
}

// Serve serves a REX server.
func Serve(ctx context.Context, config ServerConfig, onStart func(port, tlsPort uint16)) (c chan error) {
	c = make(chan error, 1)

	if tls := config.TLS; tls.AutoTLS.AcceptTOS || (tls.CertFile != "" && tls.KeyFile != "") {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		errc := make(chan error, 1)
		err := serve(ctx, &config, errc)
		if err != nil {
			c <- err
			return
		}
		err = serveTLS(ctx, &config, errc)
		if err != nil {
			c <- err
			return
		}
		if onStart != nil {
			onStart(config.Port, config.TLS.Port)
		}
		c <- <-errc
		return
	}

	errc := make(chan error, 1)
	err := serve(ctx, &config, errc)
	if err != nil {
		c <- err
		return
	}
	if onStart != nil {
		onStart(config.Port, 0)
	}
	c <- <-errc
	return
}

// Start starts a REX server.
func Start(ctx context.Context, port uint16, onStart func(port uint16)) chan error {
	c := make(chan error, 1)
	err := serve(ctx, &ServerConfig{
		Port: port,
	}, c)
	if err != nil {
		c <- err
		return c
	}
	if onStart != nil {
		onStart(port)
	}
	return c
}

// StartWithTLS starts a REX server with TLS.
func StartWithTLS(ctx context.Context, port uint16, certFile string, keyFile string, onStart func(port uint16)) chan error {
	c := make(chan error, 1)
	err := serveTLS(ctx, &ServerConfig{
		TLS: TLSConfig{
			Port:     port,
			CertFile: certFile,
			KeyFile:  keyFile,
		},
	}, c)
	if err != nil {
		c <- err
		return c
	}
	if onStart != nil {
		onStart(port)
	}
	return c
}

// StartWithAutoTLS starts a REX server with autocert powered by Let's Encrypto SSL
func StartWithAutoTLS(ctx context.Context, port uint16, onStart func(port uint16)) chan error {
	c := make(chan error, 1)
	err := serveTLS(ctx, &ServerConfig{
		TLS: TLSConfig{
			Port: port,
			AutoTLS: AutoTLSConfig{
				AcceptTOS: true,
				CacheDir:  "/var/rex/autotls",
			},
		},
	}, c)
	if err != nil {
		c <- err
		return c
	}
	if onStart != nil {
		onStart(port)
	}
	return c
}
