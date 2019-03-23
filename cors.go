package rex

import (
	"net/http"
	"strconv"
	"strings"
)

type CORS struct {
	Origin      string
	Methods     []string
	Headers     []string
	Credentials bool
	MaxAge      int // seconds
}

func PublicCORS() *CORS {
	return &CORS{
		Origin:      "*",
		Methods:     []string{"OPTIONS", "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		Headers:     []string{"Accept", "Accept-Encoding", "Accept-Lang", "Content-Type", "Authorization", "X-Requested-With"},
		Credentials: true,
		MaxAge:      60,
	}
}

func (cors *CORS) Apply(w http.ResponseWriter) {
	if len(cors.Origin) > 0 {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", cors.Origin)
		h.Set("Vary", "Origin")
		if len(cors.Methods) > 0 {
			h.Set("Access-Control-Allow-Methods", strings.Join(cors.Methods, ", "))
		}
		if len(cors.Headers) > 0 {
			h.Set("Access-Control-Allow-Headers", strings.Join(cors.Headers, ", "))
		}
		if cors.Credentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		if cors.MaxAge > 0 {
			h.Set("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
		}
	}
}
