package webx

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ije/gox/utils"
	"github.com/ije/webx/session"
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

	if xs.Session == nil {
		xs.Session = session.NewMemorySessionManager(time.Hour / 2)
	}

	app, err := initApp(appRoot)
	if err != nil {
		fmt.Println("server shutdown: init app failed:", err)
		return
	}
	xs.App = app

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
	utils.CatchExit(func() {
		if xs.App.debugProcess != nil {
			xs.App.debugProcess.Kill()
		}
		serv.Shutdown(nil)
	})

	err = serv.ListenAndServe()
	if err != nil {
		fmt.Println("server shutdown:", err)
	}
}
