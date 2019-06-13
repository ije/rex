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

	rest.Use(rex.Header("Server", "nginx"))

	rest.NotFound(rex.Static("../static/root", "e404.html"))

	rest.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
