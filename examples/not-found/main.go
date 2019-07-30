package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/hello/World">Say hello!</a></p>
`

func main() {
	rex.Use(rex.Header("Server", "nginx"))

	rex.NotFound(func(ctx *rex.Context) {
		ctx.Ok("Boom!")
	})

	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Start(8080)
}
