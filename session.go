package rex

import (
	"github.com/ije/rex/session"
)

// SessionStub is a stub for session
type SessionStub struct {
	session.Session
}

// SID returns the sid
func (s *SessionStub) SID() string {
	return s.Session.SID()
}

// Has checks a value exists
func (s *SessionStub) Has(key string) bool {
	ok, err := s.Session.Has(key)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
	return ok
}

// Get returns a session value
func (s *SessionStub) Get(key string) []byte {
	value, err := s.Session.Get(key)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
	return value
}

// Set sets a session value
func (s *SessionStub) Set(key string, value []byte) {
	err := s.Session.Set(key, value)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
}

// Delete removes a session value
func (s *SessionStub) Delete(key string) {
	err := s.Session.Delete(key)
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
}

// Flush flushes all session values
func (s *SessionStub) Flush() {
	err := s.Session.Flush()
	if err != nil {
		panic(&invalid{500, err.Error()})
	}
}
