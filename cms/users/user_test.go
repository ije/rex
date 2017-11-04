package user

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	P1 Privileges = 1 << (iota + 1)
	P2
	P3
)

type testManager struct {
	t      *testing.T
	lock   sync.RWMutex
	users  []*User
	indexs map[int]*User
}

func (m *testManager) GetAll() (users []*User, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	users = m.users
	return
}

func (m *testManager) GetGroup(privileges Privileges) (users []*User, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, user := range m.users {
		if user.Privileges&privileges != 0 {
			users = append(users, user)
		}
	}
	return
}

func (m *testManager) Get(v interface{}) (user *User, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if id, ok := v.(int); ok {
		user, _ = m.indexs[id]
	}
	return
}

func (m *testManager) CheckLoginToken(token string) (user *User, err error) {
	m.t.Log("CheckLoginToken", token)
	return
}

func (m *testManager) Add(user User, passwordHash string) (uid int, err error) {
	m.t.Log("Add", user, passwordHash)

	m.lock.Lock()
	uid = len(m.users) + 1
	m.lock.Unlock()

	_user := &user
	_user.ID = uid
	_user.Joined = time.Now()

	m.lock.Lock()
	m.users = append(m.users, _user)
	m.indexs[uid] = _user
	m.lock.Unlock()
	return
}

func (m *testManager) Update(uid int, changes map[string]interface{}) (err error) {
	m.t.Log("Update", uid, changes)

	user, err := m.Get(uid)
	if err != nil || user == nil {
		return
	}

	for key, val := range changes {
		switch strings.ToLower(key) {
		case "loginid":
			if s, ok := val.(string); ok && len(s) > 0 {
				user.LoginID = s
			}
		case "name":
			if s, ok := val.(string); ok && len(s) > 0 {
				user.Name = s
			}
		case "privileges":
			if p, ok := val.(Privileges); ok {
				user.Privileges = p
			}
		case "logined":
			if t, ok := val.(time.Time); ok {
				user.Logined = t
			}
		case "meta":
			if m, ok := val.(map[string]interface{}); ok {
				user.Meta = m
			}
		}
	}
	return
}

func (m *testManager) UpdatePassword(uid int, passwordHash string) error {
	m.t.Log("UpdatePassword", uid, passwordHash)
	return nil
}

func (m *testManager) UpdateLoginToken(uid int, token string) error {
	m.t.Log("SaveLoginToken", uid, token)
	return nil
}

func (m *testManager) Remove(uids ...int) (err error) {
	m.t.Log("Remove", uids)

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, uid := range uids {
		var users []*User
		for _, user := range m.users {
			if user.ID != uid {
				users = append(users, user)
			}
		}
		m.users = users
		delete(m.indexs, uid)
	}

	return
}

func (m *testManager) MatchPassword(uid int, passwordHash string) (match bool, err error) {
	m.t.Log("MatchPassword", uid, passwordHash)
	return false, nil
}

func (m *testManager) LogAction(uid int, action string, kind string, kindId string, extra map[string]interface{}) error {
	m.t.Log("LogAction", uid, action, kind, kindId, extra)
	return nil
}

func TestManager(t *testing.T) {
	var manager Manager
	manager = &testManager{t: t, indexs: map[int]*User{}}
	for i := 1; i < 4; i++ {
		manager.Add(User{
			LoginID:    fmt.Sprintf("user_%d", i),
			Name:       fmt.Sprintf("user#%d", i),
			Privileges: Privileges(1 << uint(i)),
			Logined:    time.Now(),
			Meta: map[string]interface{}{
				"avatar": fmt.Sprintf("avatars/user_%d.png", i),
			},
		}, "12345678")
	}
	t.Log(manager.GetAll())
	t.Log(manager.GetGroup(P1))
	t.Log(manager.GetGroup(P2 | P3))
	t.Log(manager.GetGroup(P1 | P2 | P3))
	t.Log(manager.Get(1))
	manager.CheckLoginToken("token_xxxxxx")
	manager.Remove(1)
	t.Log(manager.GetAll())
	manager.Remove(2, 3)
	t.Log(manager.GetAll())
}
