package webx

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ije/gox/utils"
)

var config = &ServerConfig{}

type ServerConfig struct {
	Port              uint16
	ReadTimeout       int
	WriteTimeout      int
	MaxHeaderBytes    int
	CustomHttpHeaders map[string]string
	Debug             bool
}

func Serve(appRoot string, serverConfig *ServerConfig) {
	if serverConfig != nil {
		config = serverConfig
	}
	if config.Port == 0 {
		config.Port = 80
	}
	if ev := os.Getenv("WEBX_DEBUG"); ev == "1" || ev == "true" {
		config.Debug = true
	}

	if appRoot != "" {
		fi, err := os.Lstat(appRoot)
		if err == nil && !fi.IsDir() {
			err = errf("invalid directory")
		}
		if err != nil {
			fmt.Printf("incorrect appRoot '%s': %v\n", appRoot, err)
			return
		}

		app, err := initApp(appRoot)
		if err != nil {
			fmt.Println("initialize app failed:", err)
			return
		}
		xs.App = app
	}

	if !config.Debug {
		xs.Log.SetLevelByName("info")
		xs.Log.SetQuite(true)
	}

	serv := &http.Server{
		Addr:           strf((":%d"), config.Port),
		Handler:        &HttpServerMux{},
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	go func() {
		err := serv.ListenAndServe()
		if err != nil {
			fmt.Println("server shutdown:", err)
		}
		os.Exit(1)
	}()

	utils.WaitExit(func(signal os.Signal) bool {
		if xs.App.debugProcess != nil {
			xs.App.debugProcess.Kill()
		}
		serv.Shutdown(nil)
		return true
	})
}
