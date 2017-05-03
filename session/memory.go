package session

import (
	"sync"
	"time"

	"github.com/ije/gox/crypto/rs"
)

type MemorySession struct {
	sid     string
	values  map[string]interface{}
	expires time.Time
}

func (ms *MemorySession) SID() string {
	return ms.sid
}

func (ms *MemorySession) Values() map[string]interface{} {
	return ms.values
}

func (ms *MemorySession) Get(key string) (value interface{}, ok bool) {
	if ms.values == nil {
		return
	}

	value, ok = ms.values[key]
	return
}

func (ms *MemorySession) Set(key string, value interface{}) {
	if ms.values == nil {
		ms.values = map[string]interface{}{}
	}
	ms.values[key] = value
}

func (ms *MemorySession) Delete(key string) {
	if ms.values == nil {
		return
	}

	delete(ms.values, key)
	return
}

func (ms *MemorySession) Flush() {
	ms.values = map[string]interface{}{}
	return
}

type MemorySessionManager struct {
	lock       sync.RWMutex
	sessions   map[string]*MemorySession
	gcLifetime time.Duration
	gcTicker   *time.Ticker
}

func NewMemorySessionManager(gcLifetime time.Duration) *MemorySessionManager {
	sp := &MemorySessionManager{
		sessions: map[string]*MemorySession{},
	}
	sp.SetGCLifetime(gcLifetime)
	go func(sp *MemorySessionManager) {
		for {
			sp.lock.RLock()
			ticker := sp.gcTicker
			sp.lock.RUnlock()

			if ticker != nil {
				<-ticker.C
				sp.GC()
			}
		}
	}(sp)

	return sp
}

func (sp *MemorySessionManager) SetGCLifetime(lifetime time.Duration) error {
	if lifetime < time.Second {
		return nil
	}

	sp.lock.Lock()
	defer sp.lock.Unlock()

	if sp.gcTicker != nil {
		sp.gcTicker.Stop()
	}
	sp.gcLifetime = lifetime
	sp.gcTicker = time.NewTicker(lifetime)

	return nil
}

func (sp *MemorySessionManager) Get(sid string) (session Session, err error) {
	now := time.Now()
	ok := len(sid) == 64

	var ms *MemorySession
	if ok {
		sp.lock.RLock()
		ms, ok = sp.sessions[sid]
		sp.lock.RUnlock()
	}

	if ok && ms.expires.Before(now) {
		sp.lock.Lock()
		delete(sp.sessions, sid)
		sp.lock.Unlock()

		ms = nil
	}

	if ms == nil {
	NEWSID:
		sid = rs.Base64.String(64)
		sp.lock.RLock()
		_, ok := sp.sessions[sid]
		sp.lock.RUnlock()
		if ok {
			goto NEWSID
		}

		ms = &MemorySession{
			sid:    sid,
			values: map[string]interface{}{},
		}
		sp.lock.Lock()
		sp.sessions[sid] = ms
		sp.lock.Unlock()
	}

	sp.lock.Lock()
	ms.expires = now.Add(sp.gcLifetime)
	sp.lock.Unlock()

	session = ms
	return
}

func (sp *MemorySessionManager) PutBack(session Session) error {
	sp.lock.RLock()
	sess, ok := sp.sessions[session.SID()]
	sp.lock.RUnlock()

	if !ok {
		return nil
	}

	sp.lock.Lock()
	sess.values = session.Values()
	sess.expires = time.Now().Add(sp.gcLifetime)
	sp.lock.Unlock()
	return nil
}

func (sp *MemorySessionManager) Destroy(sid string) error {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	delete(sp.sessions, sid)
	return nil
}

func (sp *MemorySessionManager) GC() error {
	now := time.Now()

	sp.lock.RLock()
	defer sp.lock.RUnlock()

	for sid, session := range sp.sessions {
		if session.expires.Before(now) {
			sp.lock.RUnlock()
			sp.lock.Lock()
			delete(sp.sessions, sid)
			sp.lock.Unlock()
			sp.lock.RLock()
		}
	}

	return nil
}

func init() {
	var _ Manager = (*MemorySessionManager)(nil)
}
