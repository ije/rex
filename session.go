package rex

import (
	"net/http"
	"time"

	"github.com/ije/rex/session"
)

var defaultSessionManager = &SessionManager{
	SIDStore: &defaultSIDtore{},
	Pool:     session.NewMemorySessionPool(time.Hour / 2),
}

type SessionManager struct {
	SIDStore SessionSIDStore
	Pool     session.Pool
}

type SessionSIDStore interface {
	Get(ctx *Context) string
	Set(ctx *Context, sid string)
}

type defaultSIDtore struct{}

func (s *defaultSIDtore) Get(ctx *Context) string {
	cookie, err := ctx.GetCookie("x-session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *defaultSIDtore) Set(ctx *Context, sid string) {
	ctx.SetCookie(&http.Cookie{
		Name:     "x-session",
		Value:    sid,
		HttpOnly: true,
	})
}

type ContextSession struct {
	sess session.Session
}

func (s *ContextSession) SID() string {
	return s.sess.SID()
}

func (s *ContextSession) Has(key string) bool {
	ok, err := s.sess.Has(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return ok
}

func (s *ContextSession) Get(key string) interface{} {
	value, err := s.sess.Get(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return value
}

func (s *ContextSession) Set(key string, value interface{}) {
	err := s.sess.Set(key, value)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

func (s *ContextSession) Delete(key string) {
	err := s.sess.Delete(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

func (s *ContextSession) Flush() {
	err := s.sess.Flush()
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}
