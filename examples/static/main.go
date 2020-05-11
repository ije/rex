package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/www">WWW</a></p>
`

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Static("/www/", "./www", "e404.html")

	rex.Use(rex.SendError())
	rex.Start(8080)
}
