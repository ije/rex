package session

import (
	"time"
)

type Session interface {
	SID() string
	Values(keys ...string) (values map[string]interface{}, err error)
	Has(key string) (ok bool, err error)
	Get(key string) (value interface{}, err error)
	Set(key string, value interface{}) error
	Delete(key string) error
	Flush() error
}

type Manager interface {
	Get(sid string) (Session, error)
	Destroy(sid string) error
	GC() error
	SetGCInterval(d time.Duration) error
}
