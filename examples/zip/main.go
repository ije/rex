package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p>Download the <a href="/nil.zip">nil.zip</a></p>
<p>Download the <a href="/main.js.zip">main.js.zip</a></p>
<p>Download the <a href="/root.zip">root.zip</a></p>
`

func main() {
	rex.Use(rex.Header("Server", "nginx"))

	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Get("/nil.zip", func(ctx *rex.Context) { ctx.Zip("../static/root/nil") })
	rex.Get("/main.js.zip", func(ctx *rex.Context) { ctx.Zip("../static/root/main.js") })
	rex.Get("/root.zip", func(ctx *rex.Context) { ctx.Zip("../static/root") })

	rex.Start(8080)
}
