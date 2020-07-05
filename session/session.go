package session

// Session for http server
type Session interface {
	SID() string
	Has(key string) (ok bool, err error)
	Get(key string) (value []byte, err error)
	Set(key string, value []byte) error
	Delete(key string) error
	Flush() error
}

// A Pool to handle sessions
type Pool interface {
	GetSession(sid string) (Session, error)
	Destroy(sid string) error
}
