package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/hello/World">Say hello!</a></p>
`

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Get("/hello/:name", func(ctx *rex.Context) {
		ctx.Ok("Hello, " + ctx.URL.Param("name") + "!")
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}
