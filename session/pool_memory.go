package session

import (
	"sync"
	"time"

	"github.com/ije/gox/crypto/rs"
)

type MemorySession struct {
	lock    sync.RWMutex
	store   map[string][]byte
	sid     string
	expires time.Time
}

// SID returns the sid
func (ms *MemorySession) SID() string {
	return ms.sid
}

// Has checks a value exists
func (ms *MemorySession) Has(key string) (ok bool, err error) {
	ms.lock.RLock()
	_, ok = ms.store[key]
	ms.lock.RUnlock()

	return
}

// Get returns a session value
func (ms *MemorySession) Get(key string) (value []byte, err error) {
	ms.lock.RLock()
	value = ms.store[key]
	ms.lock.RUnlock()

	return
}

// Set sets a session value
func (ms *MemorySession) Set(key string, value []byte) error {
	ms.lock.Lock()
	ms.store[key] = value
	ms.lock.Unlock()

	return nil
}

// Delete removes a session value
func (ms *MemorySession) Delete(key string) error {
	ms.lock.Lock()
	delete(ms.store, key)
	ms.lock.Unlock()

	return nil
}

// Flush flushes all session values
func (ms *MemorySession) Flush() error {
	ms.lock.Lock()
	ms.store = map[string][]byte{}
	ms.lock.Unlock()

	return nil
}

type MemorySessionPool struct {
	lock     sync.RWMutex
	sessions map[string]*MemorySession
	ttl      time.Duration
}

// NewMemorySessionPool returns a new MemorySessionPool
func NewMemorySessionPool(lifetime time.Duration) *MemorySessionPool {
	pool := &MemorySessionPool{
		sessions: map[string]*MemorySession{},
		ttl:      lifetime,
	}
	if lifetime > time.Second {
		go pool.gcLoop()
	}
	return pool
}

// GetSession returns a session by sid
func (pool *MemorySessionPool) GetSession(sid string) (session Session, err error) {
	pool.lock.RLock()
	ms, ok := pool.sessions[sid]
	pool.lock.RUnlock()

	now := time.Now()
	if ok && ms.expires.Before(now) {
		pool.lock.Lock()
		delete(pool.sessions, sid)
		pool.lock.Unlock()
		ok = false
	}

	if !ok {
	RE:
		sid = rs.Base64.String(64)
		pool.lock.RLock()
		_, ok := pool.sessions[sid]
		pool.lock.RUnlock()
		if ok {
			goto RE
		}

		ms = &MemorySession{
			sid:     sid,
			expires: now.Add(pool.ttl),
			store:   map[string][]byte{},
		}
		pool.lock.Lock()
		pool.sessions[sid] = ms
		pool.lock.Unlock()
	} else {
		ms.expires = now.Add(pool.ttl)
	}

	session = ms
	return
}

// Destroy destroys a session by sid
func (pool *MemorySessionPool) Destroy(sid string) error {
	pool.lock.Lock()
	delete(pool.sessions, sid)
	pool.lock.Unlock()

	return nil
}

func (pool *MemorySessionPool) gcLoop() {
	t := time.NewTicker(pool.ttl)
	for {
		<-t.C
		pool.gc()
	}
}

func (pool *MemorySessionPool) gc() error {
	now := time.Now()

	pool.lock.RLock()
	defer pool.lock.RUnlock()

	for sid, session := range pool.sessions {
		if session.expires.Before(now) {
			pool.lock.RUnlock()
			pool.lock.Lock()
			delete(pool.sessions, sid)
			pool.lock.Unlock()
			pool.lock.RLock()
		}
	}

	return nil
}

func init() {
	var _ Pool = (*MemorySessionPool)(nil)
}
