package webx

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	logx "github.com/ije/gox/log"
	"github.com/ije/gox/utils"
)

var config = &ServerConfig{}
var log = &logx.Logger{}
var apisMux = &ApisMux{}

type ServerConfig struct {
	AppRoot           string            `json:"appRoot"`
	Port              uint16            `json:"port"`
	CustomHTTPHeaders map[string]string `json:"customHTTPHeaders"`
	HostRedirect      string            `json:"hostRedirect"`
	SessionCookieName string            `json:"sessionCookieName"`
	ReadTimeout       int               `json:"readTimeout"`
	WriteTimeout      int               `json:"writeTimeout"`
	MaxHeaderBytes    int               `json:"maxHeaderBytes"`
	AccessLog         string            `json:"accessLog"`
	Debug             bool              `json:"debug"`
}

func SetLogger(logger *logx.Logger) {
	if logger != nil {
		log = logger
	}
}

func Serve(serverConfig *ServerConfig) {
	if serverConfig != nil {
		config = serverConfig
	}
	if config.Port == 0 {
		config.Port = 80
	}

	if log == nil {
		log = &logx.Logger{}
	}
	if !config.Debug {
		log.SetLevelByName("info")
		log.SetQuite(true)
	}

	var app *App
	if len(config.AppRoot) > 0 {
		fi, err := os.Lstat(config.AppRoot)
		if err == nil && !fi.IsDir() {
			err = fmt.Errorf("invalid directory")
		}
		if err != nil {
			log.Errorf("initialize app: incorrect root '%s': %v", config.AppRoot, err)
			return
		}

		app, err = initApp(config.AppRoot)
		if err != nil {
			log.Error("initialize app:", err)
			return
		}
	}
	apisMux.InitRouter(app)

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
		if app.debugProcess != nil {
			app.debugProcess.Signal(syscall.SIGTERM)
		}
		serv.Shutdown(nil)
		return false // exit main process by shutdown the http server
	})
}
