package rex

import (
	"net/http"
	"time"

	"github.com/ije/gox/valid"
	"github.com/ije/rex/session"
)

type sessionManager struct {
	sidStore SIDStore
	pool     session.Pool
}

var defaultSessionManager = &sessionManager{
	sidStore: &CookieSIDtore{},
	pool:     session.NewMemorySessionPool(time.Hour / 2),
}

type SIDStore interface {
	Get(ctx *Context) string
	Put(ctx *Context, sid string)
}

type CookieSIDtore struct {
	CookieName string
}

func (s *CookieSIDtore) cookieName() string {
	name := "x-session"
	if valid.IsSlug(s.CookieName) {
		name = s.CookieName
	}
	return name
}

func (s *CookieSIDtore) Get(ctx *Context) string {
	cookie, err := ctx.GetCookie(s.cookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *CookieSIDtore) Put(ctx *Context, sid string) {
	ctx.SetCookie(&http.Cookie{
		Name:     s.cookieName(),
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
