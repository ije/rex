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
	manager   *MemorySessionManager
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
	ms.expires = time.Now().Add(ms.manager.lifetime)
}

type MemorySessionManager struct {
	lock     sync.RWMutex
	sessions map[string]*MemorySession
	lifetime time.Duration
	gcTimer  *time.Timer
}

func NewMemorySessionManager(lifetime time.Duration) *MemorySessionManager {
	manager := &MemorySessionManager{
		sessions: map[string]*MemorySession{},
	}
	manager.SetLifetime(lifetime)
	return manager
}

func (manager *MemorySessionManager) CookieName() string {
	return ""
}

func (manager *MemorySessionManager) GetSession(sid string) (session Session, err error) {
	now := time.Now()

	manager.lock.RLock()
	ms, ok := manager.sessions[sid]
	manager.lock.RUnlock()

	if ok && ms.expires.Before(now) {
		manager.lock.Lock()
		delete(manager.sessions, sid)
		manager.lock.Unlock()
		ms, ok = nil, false
	}

	if !ok {
		if len(sid) != 64 {
		RE:
			sid = rs.Base64.String(64)
			manager.lock.RLock()
			_, ok := manager.sessions[sid]
			manager.lock.RUnlock()
			if ok {
				goto RE
			}
		}

		ms = &MemorySession{
			sid:     sid,
			expires: now.Add(manager.lifetime),
			store:   map[string]interface{}{},
			manager: manager,
		}
		manager.lock.Lock()
		manager.sessions[sid] = ms
		manager.lock.Unlock()
	} else {
		manager.lock.Lock()
		ms.expires = now.Add(manager.lifetime)
		manager.lock.Unlock()
	}

	session = ms
	return
}

func (manager *MemorySessionManager) SetLifetime(lifetime time.Duration) error {
	if lifetime < time.Second {
		return nil
	}

	manager.lifetime = lifetime
	manager.gcLoop()

	return nil
}

func (manager *MemorySessionManager) GC() error {
	now := time.Now()

	manager.lock.RLock()
	defer manager.lock.RUnlock()

	for sid, session := range manager.sessions {
		if session.expires.Before(now) {
			manager.lock.RUnlock()
			manager.lock.Lock()
			delete(manager.sessions, sid)
			manager.lock.Unlock()
			manager.lock.RLock()
		}
	}

	return nil
}

func (manager *MemorySessionManager) gcLoop() {
	if manager.gcTimer != nil {
		manager.gcTimer.Stop()
	}
	manager.gcTimer = time.AfterFunc(manager.lifetime, func() {
		manager.gcTimer = nil
		manager.GC()
		manager.gcLoop()
	})
}

func (manager *MemorySessionManager) Destroy(sid string) error {
	manager.lock.Lock()
	delete(manager.sessions, sid)
	manager.lock.Unlock()

	return nil
}

func init() {
	var _ Manager = (*MemorySessionManager)(nil)
}
