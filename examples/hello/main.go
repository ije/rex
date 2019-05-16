package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/hello/World">Say hello!</a></p>
`

func main() {
	rest := rex.New()

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html([]byte(indexHTML))
	})

	rest.Get("/hello/:name", func(ctx *rex.Context) {
		ctx.Ok("Hello, " + ctx.URL.Param("name") + "!")
	})

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
