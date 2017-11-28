package session

import (
	"sync"
	"time"

	"github.com/ije/gox/utils"

	"github.com/ije/gox/crypto/rs"
)

type MemorySession struct {
	sid       string
	storeLock sync.RWMutex
	store     map[string]interface{}
	expires   time.Time
	manager   *MemorySessionManager
}

func (ms *MemorySession) SID() string {
	return ms.sid
}

func (ms *MemorySession) Values(keys ...string) (values map[string]interface{}, err error) {
	ms.storeLock.RLock()
	if len(ms.store) > 0 {
		values = map[string]interface{}{}
		for key, value := range ms.store {
			if len(keys) == 0 || utils.ContainsString(keys, key) {
				values[key] = value
			}
		}
	}
	ms.storeLock.RUnlock()

	ms.activate()
	return
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
	ms.expires = time.Now().Add(ms.manager.gcLifetime)
}

type MemorySessionManager struct {
	lock       sync.RWMutex
	sessions   map[string]*MemorySession
	gcLifetime time.Duration
	gcTimer    *time.Timer
}

func NewMemorySessionManager(gcLifetime time.Duration) *MemorySessionManager {
	manager := &MemorySessionManager{
		sessions: map[string]*MemorySession{},
	}
	manager.SetGCLifetime(gcLifetime)

	return manager
}

func (manager *MemorySessionManager) Get(sid string) (session Session, err error) {
	now := time.Now()
	ok := len(sid) == 64

	var ms *MemorySession
	if ok {
		manager.lock.RLock()
		ms, ok = manager.sessions[sid]
		manager.lock.RUnlock()
	}

	if ok && ms.expires.Before(now) {
		manager.lock.Lock()
		delete(manager.sessions, sid)
		manager.lock.Unlock()

		ms = nil
	}

	if ms == nil {
	NEWSID:
		sid = rs.Base64.String(64)
		manager.lock.RLock()
		_, ok := manager.sessions[sid]
		manager.lock.RUnlock()
		if ok {
			goto NEWSID
		}

		ms = &MemorySession{
			sid:     sid,
			store:   map[string]interface{}{},
			manager: manager,
		}
		manager.lock.Lock()
		manager.sessions[sid] = ms
		manager.lock.Unlock()
	}

	manager.lock.Lock()
	ms.expires = now.Add(manager.gcLifetime)
	manager.lock.Unlock()

	session = ms
	return
}

func (manager *MemorySessionManager) Destroy(sid string) error {
	manager.lock.Lock()
	delete(manager.sessions, sid)
	manager.lock.Unlock()

	return nil
}

func (manager *MemorySessionManager) SetGCLifetime(lifetime time.Duration) error {
	if lifetime < time.Second {
		return nil
	}

	manager.gcLifetime = lifetime
	manager.gcTime()

	return nil
}

func (manager *MemorySessionManager) gcTime() {
	if manager.gcTimer != nil {
		manager.gcTimer.Stop()
	}

	manager.gcTimer = time.AfterFunc(manager.gcLifetime, func() {
		manager.GC()
		manager.gcTime()
	})
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

func init() {
	var _ Manager = (*MemorySessionManager)(nil)
}
