package rex

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
)

type Config struct {
	Port              uint16            `json:"port"`
	Root              string            `json:"root"`
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
	if config.AccessLogger != nil {
		config.AccessLogger.SetQuite(true)
	}

	logger := &log.Logger{}
	if config.ErrorLogger != nil {
		logger = config.ErrorLogger
	}
	if !config.Debug {
		logger.SetLevelByName("info")
		logger.SetQuite(true)
	}

	mux := &Mux{
		Root:              config.Root,
		Debug:             config.Debug,
		ServerName:        config.ServerName,
		CustomHTTPHeaders: config.CustomHTTPHeaders,
		SessionCookieName: config.SessionCookieName,
		HostRedirectRule:  config.HostRedirectRule,
		SessionManager:    session.NewMemorySessionManager(time.Hour / 2),
		NotFoundHandler:   config.NotFoundHandler,
		Logger:            logger,
		AccessLogger:      config.AccessLogger,
	}

	for _, apis := range globalAPIServices {
		mux.RegisterAPIService(apis)
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
}
