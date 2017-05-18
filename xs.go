package webx

import (
	"time"

	"github.com/ije/gox/log"
	"github.com/ije/webx/session"
	"github.com/ije/webx/user"
)

var xs = &XService{
	Session: session.NewMemorySessionManager(time.Hour / 2),
	Log:     &log.Logger{},
}

type XService struct {
	App     *App
	Session session.Manager
	Users   user.Manager
	Log     *log.Logger
}

func (xs *XService) clone() *XService {
	return &XService{xs.App, xs.Session, xs.Users, xs.Log}
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
