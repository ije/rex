package rex

import (
	"sync"
)

type State struct {
	lock  sync.RWMutex
	state map[string]interface{}
}

func (s *State) Get(key string) (v interface{}) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.state[key]
}

func (s *State) Add(key string, v interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.state[key] = v
}

func (s *State) Set(key string, v interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.state[key] = v
}

func (s *State) Del(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.state, key)
}

func NewState() *State {
	return &State{
		state: map[string]interface{}{},
	}
}
