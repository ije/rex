package main

import (
	"fmt"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/admin">Admin</a> (name: 'test', password: 'test')</p>
`

func main() {
	rest := rex.New()

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html(indexHTML)
	})

	rest.Get("/admin", rex.BasicAuth("rex", func(name string, password string) (bool, error) {
		return name == "test" && password == "test", nil
	}), func(ctx *rex.Context) {
		ctx.Ok(fmt.Sprintf("Hello, %s/%s!", ctx.BasicAuthUser().Name, ctx.BasicAuthUser().Password))
	})

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
