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
<p>404 - not found</p>
`

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.NotFound(func(ctx *rex.Context) {
		ctx.HTML(e404HTML)
	})

	rex.Start(8080)
}
