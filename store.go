package rex

import (
	"sync"
)

// A Store to store values.
type Store struct {
	values sync.Map
}

// Get returns the value stored in the store for a key, or nil if no
// value is present.
func (s *Store) Get(key string) (interface{}, bool) {
	return s.values.Load(key)
}

// Set sets the value for a key.
func (s *Store) Set(key string, value interface{}) {
	s.values.Store(key, value)
}
