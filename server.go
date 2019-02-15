package webx

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/gox/utils"
	"github.com/ije/wsx/session"
)

type ServerConfig struct {
	AppRoot           string            `json:"appRoot"`
	Port              uint16            `json:"port"`
	CustomHTTPHeaders map[string]string `json:"customHTTPHeaders"`
	HostRedirect      string            `json:"hostRedirect"`
	SessionCookieName string            `json:"sessionCookieName"`
	ReadTimeout       int               `json:"readTimeout"`
	WriteTimeout      int               `json:"writeTimeout"`
	MaxHeaderBytes    int               `json:"maxHeaderBytes"`
	ErrorLog          string            `json:"errorLog"`
	AccessLog         string            `json:"accessLog"`
	AppBuildLog       string            `json:"appBuildLog"`
	Debug             bool              `json:"debug"`
}

func Serve(config *ServerConfig, apiss ...*APIService) {
	if config == nil {
		config = &ServerConfig{}
	}
	if config.Port == 0 {
		config.Port = 80
	}

	var logger *log.Logger
	if len(config.ErrorLog) > 0 {
		logger, _ = log.New("file:" + strings.TrimPrefix(config.ErrorLog, "file:"))
	}
	if logger == nil {
		logger = &log.Logger{}
	}
	if !config.Debug {
		logger.SetLevelByName("info")
		logger.SetQuite(true)
	}

	var app *App
	if len(config.AppRoot) > 0 {
		var err error
		app, err = InitApp(config.AppRoot, config.AppBuildLog, config.Debug)
		if err != nil {
			logger.Error("initialize app failed:", err)
			return
		}
	}

	mux := &Mux{
		App:               app,
		CustomHTTPHeaders: config.CustomHTTPHeaders,
		SessionCookieName: config.SessionCookieName,
		HostRedirect:      config.HostRedirect,
		Debug:             config.Debug,
		SessionManager:    session.NewMemorySessionManager(time.Hour / 2),
		Logger:            logger,
	}
	if len(config.AccessLog) > 0 {
		var err error
		mux.AccessLogger, err = log.New("file:" + strings.TrimPrefix(config.ErrorLog, "file:"))
		if err != nil {
			logger.Error("initialize access logger:", err)
		} else {
			mux.AccessLogger.SetQuite(true)
		}
	}

	for _, apis := range apiss {
		mux.RegisterAPIService(apis)
	}

	serv := &http.Server{
		Addr:           fmt.Sprintf((":%d"), config.Port),
		Handler:        mux,
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
