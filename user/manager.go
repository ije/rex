package user

import (
	"errors"
	"strings"
	"sync"
	"time"
)

type Manager interface {
	GetAll() (users []*User, err error)
	GetGroup(privileges Privileges) (users []*User, err error)
	Get(id interface{}) (user *User, err error)
	CheckLoginToken(token string) (user *User, err error)
	Add(user User, password string) (uid int, err error)
	Update(uid int, changes map[string]interface{}) error
	UpdatePassword(uid int, password string) error
	UpdateLoginToken(uid int, token string) error
	Remove(uids ...int) error
	MatchPassword(uid int, password string) (match bool, err error)
	LogAction(uid int, action string, kind string, kindId string, extra map[string]interface{}) error
}

type CachedUsers struct {
	lock      sync.RWMutex
	users     []*User
	indexs    map[int]*User
	lidIndexs map[string]*User
	manager   Manager
}

func CacheUsers(manager Manager) (cu *CachedUsers, err error) {
	if manager == nil {
		err = errors.New("UsersDataSyncer is nil")
		return
	}

	allUsers, err := manager.GetAll()
	if err != nil {
		return
	}

	var users []*User
	var indexs = map[int]*User{}
	var lidIndexs = map[string]*User{}
	for _, user := range allUsers {
		users = append(users, user)
		indexs[user.ID] = user
		lidIndexs[user.LoginID] = user
	}

	cu = &CachedUsers{
		users:     users,
		indexs:    indexs,
		lidIndexs: lidIndexs,
		manager:   manager,
	}
	return
}

func (cu *CachedUsers) GetAll() (users []*User, err error) {
	cu.lock.RLock()
	defer cu.lock.RUnlock()

	users = cu.users
	return
}

func (cu *CachedUsers) GetGroup(privileges Privileges) (users []*User, err error) {
	cu.lock.RLock()
	defer cu.lock.RUnlock()

	for _, user := range cu.users {
		if user.Privileges&privileges != 0 {
			users = append(users, user)
		}
	}
	return
}

func (cu *CachedUsers) Get(id interface{}) (user *User, err error) {
	cu.lock.RLock()
	defer cu.lock.RUnlock()

	switch v := id.(type) {
	case int:
		user, _ = cu.indexs[v]
	case string:
		user, _ = cu.lidIndexs[v]
	}

	return
}

func (cu *CachedUsers) CheckLoginToken(token string) (user *User, err error) {
	user, err = cu.manager.CheckLoginToken(token)
	if err != nil {
		return
	}

	user, err = cu.Get(user.ID)
	return
}

func (cu *CachedUsers) Add(user User, password string) (uid int, err error) {
	uid, err = cu.manager.Add(user, password)
	if err != nil {
		return
	}

	_user := &user
	_user.ID = uid
	_user.Joined = time.Now()

	cu.lock.Lock()
	cu.users = append(cu.users, _user)
	cu.indexs[uid] = _user
	cu.lidIndexs[user.LoginID] = _user
	cu.lock.Unlock()
	return
}

func (cu *CachedUsers) Update(uid int, changes map[string]interface{}) (err error) {
	err = cu.manager.Update(uid, changes)
	if err != nil {
		return
	}

	user, err := cu.Get(uid)
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

func (cu *CachedUsers) UpdatePassword(uid int, password string) error {
	return cu.manager.UpdatePassword(uid, password)
}

func (cu *CachedUsers) UpdateLoginToken(uid int, token string) error {
	return cu.UpdateLoginToken(uid, token)
}

func (cu *CachedUsers) Remove(uids ...int) (err error) {
	err = cu.manager.Remove(uids...)
	if err != nil {
		return
	}

	cu.lock.Lock()
	defer cu.lock.Unlock()

	for _, uid := range uids {
		var ulid string
		var users []*User
		for _, user := range cu.users {
			if user.ID == uid {
				ulid = user.LoginID
			} else {
				users = append(users, user)
			}
		}
		cu.users = users
		delete(cu.indexs, uid)
		delete(cu.lidIndexs, ulid)
	}

	return
}

func (cu *CachedUsers) MatchPassword(uid int, password string) (match bool, err error) {
	return cu.manager.MatchPassword(uid, password)
}

func (cu *CachedUsers) LogAction(uid int, action string, kind string, kindId string, extra map[string]interface{}) error {
	return cu.LogAction(uid, action, kind, kindId, extra)
}

func init() {
	var _ Manager = (*CachedUsers)(nil)
}
