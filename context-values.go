package rex

import (
	"sync"
)

type ContextValues struct {
	values sync.Map
}

// Get returns the value stored in the values for a key, or nil if no
// value is present.
func (s *ContextValues) Get(key string) (interface{}, bool) {
	return s.values.Load(key)
}

// Set sets the value for a key.
func (s *ContextValues) Set(key string, value interface{}) {
	s.values.Store(key, value)
}
