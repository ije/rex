package session

// Pool interface represents a session pool.
type Pool interface {
	GetSession(sid string) (Session, error)
	Destroy(sid string) error
}
