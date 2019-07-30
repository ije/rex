package main

import (
	"fmt"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/admin">Admin</a> (name: 'rex', password: 'rex')</p>
`

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	rex.Get("/admin", rex.BasicAuth(func(name string, password string) (bool, error) {
		return name == "rex" && password == "rex", nil
	}), func(ctx *rex.Context) {
		ctx.Ok(fmt.Sprintf("Hello, %s/%s!", ctx.BasicUser().Name, ctx.BasicUser().Password))
	})

	rex.Start(8080)
}
