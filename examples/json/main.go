package main

import (
	"encoding/json"
	"net/http"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/json1">JSON #1(small)</a></p>
<p><a href="/json2">JSON #2(big)</a></p>
`

func main() {
	rest := rex.New()
	rest.Use(rex.Header("Server", "nginx"))

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html(indexHTML)
	})

	rest.Get("/json1", func(ctx *rex.Context) {
		ctx.Json(200, map[string]string{
			"foo": "bar",
		})
	})

	rest.Get("/json2", func(ctx *rex.Context) {
		resp, err := http.Get("https://api.github.com/")
		if err != nil {
			ctx.Error(err)
			return
		}

		var ret map[string]string
		err = json.NewDecoder(resp.Body).Decode(&ret)
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.Json(200, ret)
	})

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
