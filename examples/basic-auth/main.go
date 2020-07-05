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
		if name == "rex" && password == "rex" {
			return true, nil
		}
		return false, nil
	}), func(ctx *rex.Context) {
		user := ctx.BasicAuthUserName()
		ctx.Ok(fmt.Sprintf("Hello, %s!", user))
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}
