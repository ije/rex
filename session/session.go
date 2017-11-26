package session

import (
	"time"
)

type Session interface {
	SID() string
	Values(keys ...string) (values map[string]interface{}, err error)
	Get(key string) (value interface{}, ok bool, err error)
	Set(key string, value interface{}) error
	Delete(key string) error
	Flush() error
}

type Manager interface {
	Get(sid string) (Session, error)
	Destroy(sid string) error
	SetGCLifetime(lifetime time.Duration) error
	GC() error
}
