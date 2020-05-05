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

	rex.Get("/admin", rex.BasicAuth(func(name string, password string) (interface{}, error) {
		if name == "rex" && password == "rex" {
			return "rex", nil
		}
		return nil, nil
	}), func(ctx *rex.Context) {
		user, _ := ctx.BasicUser()
		ctx.Ok(fmt.Sprintf("Hello, %v!", user))
	})

	rex.Start(8080)
}
