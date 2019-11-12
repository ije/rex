package session

import (
	"net/http"

	"github.com/ije/gox/valid"
)

type SIDManager interface {
	Get(r *http.Request) string
	Put(w http.ResponseWriter, sid string)
}

type CookieSIDManager struct {
	CookieName string
}

func (s *CookieSIDManager) cookieName() string {
	name := "x-session"
	if valid.IsSlug(s.CookieName) {
		name = s.CookieName
	}
	return name
}

func (s *CookieSIDManager) Get(r *http.Request) string {
	cookie, err := r.Cookie(s.cookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *CookieSIDManager) Put(w http.ResponseWriter, sid string) {
	cookie := &http.Cookie{
		Name:     s.cookieName(),
		Value:    sid,
		HttpOnly: true,
	}
	w.Header().Add("Set-Cookie", cookie.String())
}
