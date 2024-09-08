package session

import (
	"net/http"
	"strings"
)

// A IdHandler to handle session id
type IdHandler interface {
	Get(r *http.Request) string
	Put(w http.ResponseWriter, id string)
}

// A CookieIdHandler to handle session id by http cookie header
type CookieIdHandler struct {
	cookieName string
}

// NewCookieIdHandler returns a new CookieIdHandler
func NewCookieIdHandler(cookieName string) *CookieIdHandler {
	return &CookieIdHandler{cookieName: strings.TrimSpace(cookieName)}
}

// CookieName returns cookie name
func (s *CookieIdHandler) CookieName() string {
	name := strings.TrimSpace(s.cookieName)
	if name == "" {
		name = "x-session"
	}
	return name
}

// Get return seesion id by http cookie
func (s *CookieIdHandler) Get(r *http.Request) string {
	cookie, err := r.Cookie(s.CookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Put sets seesion id by http cookie
func (s *CookieIdHandler) Put(w http.ResponseWriter, id string) {
	cookie := &http.Cookie{
		Name:     s.CookieName(),
		Value:    id,
		HttpOnly: true,
	}
	w.Header().Add("Set-Cookie", cookie.String())
}
