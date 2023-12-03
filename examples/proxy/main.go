package main

import (
	"net/http"

	"github.com/ije/rex"
)

func main() {
	// proxy jex.me
	rex.Use(func(ctx *rex.Context) interface{} {
		host := ctx.R.Host
		if host == "jex.me" || host == "jex.me:443" || host == "localhost:8087" {
			ctx.R.URL.Scheme = "http"
			ctx.R.URL.Host = "jex.me"
			resp, err := http.Get(ctx.R.URL.String())
			if err != nil {
				return err
			}
			return resp
		}
		return rex.Status(404, "Not Found")
	})

	// fmt.Println("Server running on http://localhost:8087")
	// <-rex.Start(8087)
	<-rex.StartWithAutoTLS(443, "jex.me")
}
