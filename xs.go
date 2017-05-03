package webx

import (
	"github.com/ije/gox/log"
	"github.com/ije/webx/session"
	"github.com/ije/webx/user"
)

var xs = &XService{Log: &log.Logger{}}

type XService struct {
	App     *App
	Log     *log.Logger
	Session session.Manager
	Users   user.Manager
}

func InitLogger(path string, buffer int, maxFileSize int) (err error) {
	xs.Log, err = log.New(strf("file:%s?buffer=%d&maxBytes=%d", path, buffer, maxFileSize))
	return
}

func InitSession(sessionManager session.Manager) {
	if sessionManager != nil {
		xs.Session = sessionManager
	}
}

func InitUserManager(userManager user.Manager) {
	if userManager != nil {
		xs.Users = userManager
	}
}
