package rex

import (
	"github.com/ije/rex/session"
)

type Session struct {
	sess session.Session
}

func (s *Session) SID() string {
	return s.sess.SID()
}

func (s *Session) Has(key string) bool {
	ok, err := s.sess.Has(key)
	if err != nil {
		panic(&ctxPanicError{err.Error()})
	}
	return ok
}

func (s *Session) Get(key string) interface{} {
	value, err := s.sess.Get(key)
	if err != nil {
		panic(&ctxPanicError{err.Error()})
	}
	return value
}

func (s *Session) Set(key string, value interface{}) {
	err := s.sess.Set(key, value)
	if err != nil {
		panic(&ctxPanicError{err.Error()})
	}
}

func (s *Session) Delete(key string) {
	err := s.sess.Delete(key)
	if err != nil {
		panic(&ctxPanicError{err.Error()})
	}
}

func (s *Session) Flush() {
	err := s.sess.Flush()
	if err != nil {
		panic(&ctxPanicError{err.Error()})
	}
}
