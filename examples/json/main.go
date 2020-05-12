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
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Get("/json1", func(ctx *rex.Context) {
		ctx.JSON(map[string]string{
			"foo": "bar",
		})
	})

	rex.Get("/json2", func(ctx *rex.Context) {
		resp, err := http.Get("https://api.github.com/emojis")
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadGateway)
			return
		}

		var ret map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ret)
		if err != nil {
			ctx.Error(err.Error(), 500)
			return
		}

		ctx.JSON(ret)
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}
