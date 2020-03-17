package session

import (
	"time"
)

// Session for http server
type Session interface {
	SID() string
	Has(key string) (ok bool, err error)
	Get(key string) (value interface{}, err error)
	Set(key string, value interface{}) error
	Delete(key string) error
	Flush() error
}

// A Pool to handle sessions
type Pool interface {
	GetSession(sid string) (Session, error)
	SetLifetime(d time.Duration) error
	GC() error
	Destroy(sid string) error
}
