package session

// Session interface represents a http session.
type Session interface {
	// SID returns the sid.
	SID() string
	// Has checks a value exists.
	Has(key string) (ok bool, err error)
	// Get returns a session valuen.
	Get(key string) (value []byte, err error)
	// Set sets a session value.
	Set(key string, value []byte) error
	// Delete removes a session value.
	Delete(key string) error
	// Flush flushes all session values.
	Flush() error
}
