package session

import (
	"net/http"
	"strings"
)

// A SidHandler to handle session id
type SidHandler interface {
	Get(r *http.Request) string
	Put(w http.ResponseWriter, id string)
}

// A CookieSidHandler to handle session id by http cookie header
type CookieSidHandler struct {
	cookieName string
}

// NewCookieSidHandler returns a new CookieIdHandler
func NewCookieSidHandler(cookieName string) *CookieSidHandler {
	return &CookieSidHandler{cookieName: strings.TrimSpace(cookieName)}
}

// CookieName returns cookie name
func (s *CookieSidHandler) CookieName() string {
	name := strings.TrimSpace(s.cookieName)
	if name == "" {
		name = "SID"
	}
	return name
}

// Get return seesion id by http cookie
func (s *CookieSidHandler) Get(r *http.Request) string {
	cookie, err := r.Cookie(s.CookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Put sets seesion id by http cookie
func (s *CookieSidHandler) Put(w http.ResponseWriter, id string) {
	cookie := &http.Cookie{
		Name:     s.CookieName(),
		Value:    id,
		HttpOnly: true,
	}
	w.Header().Add("Set-Cookie", cookie.String())
}
