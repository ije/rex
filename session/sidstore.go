package session

import (
	"net/http"

	"github.com/ije/gox/valid"
)

type SIDStore interface {
	Get(r *http.Request) string
	Put(w http.ResponseWriter, sid string)
}

type CookieSIDStore struct {
	CookieName string
}

func (s *CookieSIDStore) cookieName() string {
	name := "x-session"
	if valid.IsSlug(s.CookieName) {
		name = s.CookieName
	}
	return name
}

func (s *CookieSIDStore) Get(r *http.Request) string {
	cookie, err := r.Cookie(s.cookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *CookieSIDStore) Put(w http.ResponseWriter, sid string) {
	cookie := &http.Cookie{
		Name:     s.cookieName(),
		Value:    sid,
		HttpOnly: true,
	}
	w.Header().Add("Set-Cookie", cookie.String())
}
