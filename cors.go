package rex

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSHeaders struct {
	AllowOrigin      string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

func PublicCORS() *CORSHeaders {
	return &CORSHeaders{
		AllowOrigin:      "*",
		AllowMethods:     []string{"OPTIONS", "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Accept", "Accept-Encoding", "Accept-Lang", "Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           60,
	}
}

func (cors *CORSHeaders) Apply(w http.ResponseWriter) {
	if len(cors.AllowOrigin) > 0 {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", cors.AllowOrigin)
		h.Set("Vary", "Origin")
		if len(cors.AllowMethods) > 0 {
			h.Set("Access-Control-Allow-Methods", strings.Join(cors.AllowMethods, ", "))
		}
		if len(cors.AllowHeaders) > 0 {
			h.Set("Access-Control-Allow-Headers", strings.Join(cors.AllowHeaders, ", "))
		}
		if cors.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		if cors.MaxAge > 0 {
			h.Set("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
		}
	}
}
