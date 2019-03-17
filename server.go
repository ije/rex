package rex

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/gox/utils"
	"github.com/ije/rex/session"
)

type Config struct {
	Port              uint16            `json:"port"`
	AppDir            string            `json:"appDir"`
	ServerName        string            `json:"serverName"`
	CustomHTTPHeaders map[string]string `json:"customHTTPHeaders"`
	SessionCookieName string            `json:"sessionCookieName"`
	HostRedirectRule  string            `json:"hostRedirectRule"`
	ReadTimeout       uint32            `json:"readTimeout"`
	WriteTimeout      uint32            `json:"writeTimeout"`
	MaxHeaderBytes    uint32            `json:"maxHeaderBytes"`
	Debug             bool              `json:"debug"`
	AppBuildLogFile   string            `json:"appBuildLogFile"`
	ErrorLogger       *log.Logger       `json:"-"`
	AccessLogger      *log.Logger       `json:"-"`
}

func Serve(config Config) {
	if config.Port == 0 {
		config.Port = 80
	}

	logger := &log.Logger{}
	if config.ErrorLogger != nil {
		logger = config.ErrorLogger
	}
	if !config.Debug {
		logger.SetLevelByName("info")
		logger.SetQuite(true)
	}

	var app *App
	if len(config.AppDir) > 0 {
		var err error
		app, err = InitApp(config.AppDir, config.AppBuildLogFile, config.Debug)
		if err != nil {
			logger.Error("initialize app:", err)
			return
		}
	}

	mux := &Mux{
		App:               app,
		Debug:             config.Debug,
		ServerName:        config.ServerName,
		CustomHTTPHeaders: config.CustomHTTPHeaders,
		SessionCookieName: config.SessionCookieName,
		HostRedirectRule:  config.HostRedirectRule,
		SessionManager:    session.NewMemorySessionManager(time.Hour / 2),
		Logger:            logger,
	}

	for _, apis := range globalAPIServices {
		mux.RegisterAPIService(apis)
	}

	if config.AccessLogger != nil {
		mux.AccessLogger = config.AccessLogger
		mux.AccessLogger.SetQuite(true)
	}

	serv := &http.Server{
		Addr:           fmt.Sprintf((":%d"), config.Port),
		Handler:        mux,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: int(config.MaxHeaderBytes),
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
