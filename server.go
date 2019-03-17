package rex

import (
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/ije/gox/log"
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
	NotFoundHandler   http.Handler      `json:"-"`
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
		app, err = InitApp(config.AppDir, config.Debug)
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
		NotFoundHandler:   config.NotFoundHandler,
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

	err := serv.ListenAndServe()
	if err != nil {
		fmt.Println("rex server shutdown:", err)
	}

	if app != nil && app.debugProcess != nil {
		app.debugProcess.Signal(syscall.SIGTERM)
	}
}
