package rex

import (
	"sync"
)

type State struct {
	lock  sync.RWMutex
	state map[string]interface{}
}

func (s *State) Has(key string) (ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, ok = s.state[key]
	return
}

func (s *State) Get(key string) (v interface{}) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v, _ = s.state[key]
	return
}

func (s *State) Set(key string, v interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.state[key] = v
}

func (s *State) Delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.state, key)
}

func NewState() *State {
	return &State{
		state: map[string]interface{}{},
	}
}
