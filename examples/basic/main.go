package main

import (
	"fmt"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/admin">Admin</a> (user: 'test', pass: 'test')</p>
`

func main() {
	rest := rex.New()

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html(indexHTML)
	})

	rest.Get("/admin", rex.BasicAuth("rex", func(user string, pass string) (bool, error) {
		return user == "test" && pass == "test", nil
	}), func(ctx *rex.Context) {
		ctx.Ok(fmt.Sprintf("Hello, %s/%s!", ctx.BasicUser().Name, ctx.BasicUser().Pass))
	})

	rex.Serve(rex.Config{
		Port: 8080,
	})
}
