package rex

import (
	"github.com/ije/rex/session"
)

// ContextSession handles context session
type ContextSession struct {
	sess session.Session
}

// SID returns the sid
func (s *ContextSession) SID() string {
	return s.sess.SID()
}

// Has checks a value exists
func (s *ContextSession) Has(key string) bool {
	ok, err := s.sess.Has(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return ok
}

// Get returns a session value
func (s *ContextSession) Get(key string) interface{} {
	value, err := s.sess.Get(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
	return value
}

// Set sets a session value
func (s *ContextSession) Set(key string, value interface{}) {
	err := s.sess.Set(key, value)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

// Delete removes a session value
func (s *ContextSession) Delete(key string) {
	err := s.sess.Delete(key)
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}

// Flush flushes all session values
func (s *ContextSession) Flush() {
	err := s.sess.Flush()
	if err != nil {
		panic(&contextPanicError{err.Error()})
	}
}
