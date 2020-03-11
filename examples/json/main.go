package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/json1">JSON #1(small)</a></p>
<p><a href="/json2">JSON #2(big)</a></p>
<p><a href="/json3">JSON #3(500)</a></p>
<p><a href="/json4">JSON #4(400)</a></p>
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
		resp, err := http.Get("https://api.github.com/")
		if err != nil {
			ctx.Error(err)
			return
		}

		var ret map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ret)
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.JSON(ret)
	})

	rex.Get("/json3", func(ctx *rex.Context) {
		ctx.JSONError(errors.New("Boom"))
	})

	rex.Get("/json4", func(ctx *rex.Context) {
		ctx.JSONError(rex.Invalid(404, "item not found"))
	})

	rex.Start(8080)
}
