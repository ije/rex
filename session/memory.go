package session

import (
	"sync"
	"time"

	"github.com/ije/gox/crypto/rs"
)

type MemorySession struct {
	lock    sync.RWMutex
	store   map[string]interface{}
	sid     string
	expires time.Time
}

func (ms *MemorySession) SID() string {
	return ms.sid
}

func (ms *MemorySession) Has(key string) (ok bool, err error) {
	ms.lock.RLock()
	_, ok = ms.store[key]
	ms.lock.RUnlock()

	return
}

func (ms *MemorySession) Get(key string) (value interface{}, err error) {
	ms.lock.RLock()
	value, _ = ms.store[key]
	ms.lock.RUnlock()

	return
}

func (ms *MemorySession) Set(key string, value interface{}) error {
	ms.lock.Lock()
	ms.store[key] = value
	ms.lock.Unlock()

	return nil
}

func (ms *MemorySession) Delete(key string) error {
	ms.lock.Lock()
	delete(ms.store, key)
	ms.lock.Unlock()

	return nil
}

func (ms *MemorySession) Flush() error {
	ms.lock.Lock()
	ms.store = map[string]interface{}{}
	ms.lock.Unlock()

	return nil
}

type MemorySessionPool struct {
	lock     sync.RWMutex
	sessions map[string]*MemorySession
	lifetime time.Duration
}

func NewMemorySessionPool(lifetime time.Duration) *MemorySessionPool {
	pool := &MemorySessionPool{
		sessions: map[string]*MemorySession{},
		lifetime: lifetime,
	}
	if lifetime > time.Second {
		go pool.gcLoop()
	}
	return pool
}

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
			expires: now.Add(pool.lifetime),
			store:   map[string]interface{}{},
		}
		pool.lock.Lock()
		pool.sessions[sid] = ms
		pool.lock.Unlock()
	} else {
		ms.expires = now.Add(pool.lifetime)
	}

	session = ms
	return
}

func (pool *MemorySessionPool) Destroy(sid string) error {
	pool.lock.Lock()
	delete(pool.sessions, sid)
	pool.lock.Unlock()

	return nil
}

func (pool *MemorySessionPool) gcLoop() {
	t := time.Tick(pool.lifetime)
	for {
		<-t
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
