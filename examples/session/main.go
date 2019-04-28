package main

import (
	"html/template"
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
	rest.Template = template.Must(template.New("index").Parse(indexHTML))
	rest.Use(
		rex.Header("Server", "nginx"),
		rex.SessionManager(session.NewMemorySessionManager(15*time.Second)),
	)

	rest.Get("/", func(ctx *rex.Context) {
		sess := ctx.Session()
		user, _ := sess.Get("user")
		ctx.Render("index", map[string]interface{}{
			"user": user,
		})
	})

	rest.Post("/login", func(ctx *rex.Context) {
		sess := ctx.Session()
		user := ctx.FormString("user", "")
		if user != "" {
			sess.Set("user", user)
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
