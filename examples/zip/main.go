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
	rest := rex.New()
	rest.Use(rex.Header("Server", "nginx"))

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html(indexHTML)
	})

	rest.Get("/nil.zip", rex.Zip("../static/root/nil"))
	rest.Get("/main.js.zip", rex.Zip("../static/root/main.js"))
	rest.Get("/root.zip", rex.Zip("../static/root"))

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
