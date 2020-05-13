package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/hello/World">Say hello!</a></p>
`

const e404HTML = `
<h1>Welcome to use REX!</h1>
<p>404 - page not found</p>
`

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Fallback(func(ctx *rex.Context) {
		ctx.HTML(e404HTML)
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}
