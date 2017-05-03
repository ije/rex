package session

import (
	"time"
)

type Session interface {
	SID() string
	Values() map[string]interface{}
	Get(key string) (value interface{}, ok bool)
	Set(key string, value interface{})
	Delete(key string)
	Flush()
}

type Manager interface {
	Get(sid string) (Session, error)
	PutBack(session Session) error
	Destroy(sid string) error
	SetGCLifetime(lifetime time.Duration) error
	GC() error
}
