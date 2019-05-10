package main

import (
	"time"

	"github.com/ije/rex"
	"github.com/ije/rex/session"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>{{if .user}}
<p>Hello, {{.user}}</p>
<p><a href="/logout">Logout</a></p>{{else}}
<form method="post" action="/login">
	<label>Login as:</label>
	<input name="user" type="text">
</form>{{end}}
`

func main() {
	rest := rex.New()
	rest.Use(
		rex.SessionManager(session.NewMemorySessionManager(15 * time.Second)),
	)

	rest.Get("/", func(ctx *rex.Context) {
		ctx.RenderHTML(indexHTML, map[string]interface{}{
			"user": ctx.Session().Get("user"),
		})
	})

	rest.Post("/login", func(ctx *rex.Context) {
		user := ctx.FormString("user", "")
		if user != "" {
			ctx.Session().Set("user", user)
		}
		ctx.Redirect(301, "/")
	})

	rest.Get(
		"/logout",
		rex.Header("Cache-Control", "no-cache, no-store, must-revalidate"),
		func(ctx *rex.Context) {
			ctx.Session().Delete("user")
			ctx.Redirect(301, "/")
		},
	)

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
