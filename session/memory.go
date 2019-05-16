package session

import (
	"sync"
	"time"

	"github.com/ije/gox/crypto/rs"
)

type MemorySession struct {
	sid       string
	expires   time.Time
	storeLock sync.RWMutex
	store     map[string]interface{}
	pool      *MemorySessionPool
}

func (ms *MemorySession) SID() string {
	return ms.sid
}

func (ms *MemorySession) Has(key string) (ok bool, err error) {
	ms.storeLock.RLock()
	_, ok = ms.store[key]
	ms.storeLock.RUnlock()

	ms.activate()
	return
}

func (ms *MemorySession) Get(key string) (value interface{}, err error) {
	ms.storeLock.RLock()
	value, _ = ms.store[key]
	ms.storeLock.RUnlock()

	ms.activate()
	return
}

func (ms *MemorySession) Set(key string, value interface{}) error {
	ms.storeLock.Lock()
	ms.store[key] = value
	ms.storeLock.Unlock()

	ms.activate()
	return nil
}

func (ms *MemorySession) Delete(key string) error {
	ms.storeLock.Lock()
	delete(ms.store, key)
	ms.storeLock.Unlock()

	ms.activate()
	return nil
}

func (ms *MemorySession) Flush() error {
	ms.storeLock.Lock()
	ms.store = map[string]interface{}{}
	ms.storeLock.Unlock()

	ms.activate()
	return nil
}

func (ms *MemorySession) activate() {
	ms.expires = time.Now().Add(ms.pool.lifetime)
}

type MemorySessionPool struct {
	lock     sync.RWMutex
	sessions map[string]*MemorySession
	lifetime time.Duration
	gcTimer  *time.Timer
}

func NewMemorySessionPool(lifetime time.Duration) *MemorySessionPool {
	pool := &MemorySessionPool{
		sessions: map[string]*MemorySession{},
	}
	pool.SetLifetime(lifetime)
	return pool
}

func (pool *MemorySessionPool) CookieName() string {
	return ""
}

func (pool *MemorySessionPool) GetSession(sid string) (session Session, err error) {
	now := time.Now()

	pool.lock.RLock()
	ms, ok := pool.sessions[sid]
	pool.lock.RUnlock()

	if ok && ms.expires.Before(now) {
		pool.lock.Lock()
		delete(pool.sessions, sid)
		pool.lock.Unlock()
		ms, ok = nil, false
	}

	if !ok {
		if len(sid) != 64 {
		RE:
			sid = rs.Base64.String(64)
			pool.lock.RLock()
			_, ok := pool.sessions[sid]
			pool.lock.RUnlock()
			if ok {
				goto RE
			}
		}

		ms = &MemorySession{
			sid:     sid,
			expires: now.Add(pool.lifetime),
			store:   map[string]interface{}{},
			pool:    pool,
		}
		pool.lock.Lock()
		pool.sessions[sid] = ms
		pool.lock.Unlock()
	} else {
		pool.lock.Lock()
		ms.expires = now.Add(pool.lifetime)
		pool.lock.Unlock()
	}

	session = ms
	return
}

func (pool *MemorySessionPool) SetLifetime(lifetime time.Duration) error {
	if lifetime < time.Second {
		return nil
	}

	pool.lifetime = lifetime
	pool.gcLoop()

	return nil
}

func (pool *MemorySessionPool) GC() error {
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

func (pool *MemorySessionPool) gcLoop() {
	if pool.gcTimer != nil {
		pool.gcTimer.Stop()
	}
	pool.gcTimer = time.AfterFunc(pool.lifetime, func() {
		pool.gcTimer = nil
		pool.GC()
		pool.gcLoop()
	})
}

func (pool *MemorySessionPool) Destroy(sid string) error {
	pool.lock.Lock()
	delete(pool.sessions, sid)
	pool.lock.Unlock()

	return nil
}

func init() {
	var _ Pool = (*MemorySessionPool)(nil)
}
