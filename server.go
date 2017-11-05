package webx

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ije/gox/utils"
)

var config = &ServerConfig{}
var apisMux = &ApisMux{}

type ServerConfig struct {
	AppRoot           string
	Port              uint16
	CustomHTTPHeaders map[string]string
	HostRedirect      string
	SessionCookieName string
	ReadTimeout       int
	WriteTimeout      int
	MaxHeaderBytes    int
	Debug             bool
}

func Serve(serverConfig *ServerConfig) {
	if serverConfig != nil {
		config = serverConfig
	}
	if config.Port == 0 {
		config.Port = 80
	}

	if len(config.AppRoot) > 0 {
		fi, err := os.Lstat(config.AppRoot)
		if err == nil && !fi.IsDir() {
			err = fmt.Errorf("invalid directory")
		}
		if err != nil {
			fmt.Printf("incorrect AppRoot '%s': %v\n", config.AppRoot, err)
			return
		}

		app, err := initApp(config.AppRoot)
		if err != nil {
			fmt.Println("initialize app failed:", err)
			return
		}

		xs.App = app
	}

	apisMux.initRouter()

	if !config.Debug {
		xs.Log.SetLevelByName("info")
		xs.Log.SetQuite(true)
	}

	serv := &http.Server{
		Addr:           fmt.Sprintf((":%d"), config.Port),
		Handler:        apisMux,
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
		return false // exit main process by shutdown the http server
	})
}
