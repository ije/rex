package rex

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/rex/session"
)

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
		if config.AccessLogger != nil {
			config.AccessLogger.SetQuite(true)
		}
	}

	mux := &Mux{
		Config:         config,
		Logger:         logger,
		SessionManager: session.NewMemorySessionManager(time.Hour / 2),
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
