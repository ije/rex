package rex

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSOptions struct {
	AllowOrigin      string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

func PublicCORS() CORSOptions {
	return CORSOptions{
		AllowOrigin:      "*",
		AllowMethods:     []string{"OPTIONS", "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Accept", "Accept-Encoding", "Accept-Lang", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{},
		AllowCredentials: true,
		MaxAge:           60,
	}
}

func CORS(cors CORSOptions) RESTHandle {
	return func(ctx *Context) {
		if len(cors.AllowOrigin) > 0 {
			ctx.SetHeader("Vary", "Origin")
			ctx.SetHeader("Access-Control-Allow-Origin", cors.AllowOrigin)
			if cors.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if ctx.R.Method == "OPTIONS" {
				if len(cors.AllowMethods) > 0 {
					ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(cors.AllowMethods, ", "))
				}
				if len(cors.AllowHeaders) > 0 {
					ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(cors.AllowHeaders, ", "))
				}
				if cors.MaxAge > 0 {
					ctx.SetHeader("Access-Control-Max-Age", strconv.Itoa(cors.MaxAge))
				}
				ctx.End(http.StatusNoContent)
				return
			}
		}
		ctx.Next()
	}
}
