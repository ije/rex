package main

import (
	"html/template"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
{{if .user}}
	<p>Welcome back, <strong>{{.user}}</strong>!</p>

	<h2>Todos:</h2>
	<ul>
		{{range $todo := .todos}}
		<li>{{$todo}}</li>
		{{end}}
	</ul>
	<div>

	<form method="post" action="/add-todo">
		<label>Add todo:</label>
		<input name="todo" type="text">
	</form>
	</div>

	<p><a href="/logout">Logout</a></p>
{{else}}
	<form method="post" action="/login">
		<label>Login as:</label>
		<input name="user" type="text">
	</form>
{{end}}
`

type user struct {
	id          string
	permissions []string
}

func (u *user) Permissions() []string {
	return u.permissions
}

func main() {
	tpl := template.Must(template.New("").Parse(indexHTML))
	todos := map[string][]string{}

	rex.Use(func(ctx *rex.Context) {
		if ctx.Session().Has("USER") {
			ctx.SetACLUser(&user{
				id:          ctx.Session().Get("USER").(string),
				permissions: []string{"add"},
			})
		}
		ctx.Next()
	})

	rex.Get("/", func(ctx *rex.Context) {
		if u := ctx.ACLUser(); u != nil {
			ctx.Render(tpl, map[string]interface{}{
				"user":  u.(*user).id,
				"todos": todos[u.(*user).id],
			})
		} else {
			ctx.Render(tpl, nil)
		}
	})

	rex.Post("/add-todo", rex.ACL("add"), func(ctx *rex.Context) {
		todo := ctx.FormValue("todo").String()
		if todo != "" {
			user := ctx.ACLUser().(*user).id
			todos[user] = append(todos[user], todo)
		}
		ctx.Redirect("/", 301)
	})

	rex.Post("/login", func(ctx *rex.Context) {
		user := ctx.FormValue("user").String()
		if user != "" {
			ctx.Session().Set("USER", user)
		}
		ctx.Redirect("/", 301)
	})

	rex.Get(
		"/logout",
		rex.Header("Cache-Control", "no-cache, no-store, must-revalidate"),
		func(ctx *rex.Context) {
			ctx.Session().Delete("USER")
			ctx.Redirect("/", 301)
		},
	)

	rex.Start(8080)
}
