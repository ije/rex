package session

import (
	"net/http"
	"strings"
)

// A SIDStore to store sid
type SIDStore interface {
	Get(r *http.Request) string
	Put(w http.ResponseWriter, sid string)
}

// A CookieSIDStore to store sid by http cookie
type CookieSIDStore struct {
	cookieName string
}

// NewCookieSIDStore returns a new CookieSIDStore
func NewCookieSIDStore(cookieName string) *CookieSIDStore {
	return &CookieSIDStore{cookieName: strings.TrimSpace(cookieName)}
}

// CookieName returns cookie name
func (s *CookieSIDStore) CookieName() string {
	name := strings.TrimSpace(s.cookieName)
	if name == "" {
		name = "x-session"
	}
	return name
}

// Get return sid by http cookie
func (s *CookieSIDStore) Get(r *http.Request) string {
	cookie, err := r.Cookie(s.CookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Put sets sid by http cookie
func (s *CookieSIDStore) Put(w http.ResponseWriter, sid string) {
	cookie := &http.Cookie{
		Name:     s.CookieName(),
		Value:    sid,
		HttpOnly: true,
	}
	w.Header().Add("Set-Cookie", cookie.String())
}
