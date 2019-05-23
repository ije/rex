package main

import (
	"html/template"

	"github.com/ije/rex"
	"github.com/ije/rex/acl"
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

type User struct {
	id         string
	privileges []string
}

func (u *User) Privileges() []string {
	return u.privileges
}

func main() {
	rest := rex.New()
	tpl := template.Must(template.New("").Parse(indexHTML))
	todos := map[string][]string{}

	rest.Use(rex.ACLAuth(func(ctx *rex.Context) (acl.User, error) {
		if ctx.Session().Has("USER") {
			return &User{
				id:         ctx.Session().Get("USER").(string),
				privileges: []string{"add"},
			}, nil
		}
		return nil, nil
	}))

	rest.Get("/", func(ctx *rex.Context) {
		if user := ctx.ACLUser(); user != nil {
			ctx.Render(tpl, map[string]interface{}{
				"user":  user.(*User).id,
				"todos": todos[user.(*User).id],
			})
		} else {
			ctx.Render(tpl, nil)
		}
	})

	rest.Post("/add-todo", rex.Privileges("add"), func(ctx *rex.Context) {
		todo := ctx.FormValue("todo").String()
		if todo != "" {
			user := ctx.ACLUser().(*User).id
			todos[user] = append(todos[user], todo)
		}
		ctx.Redirect(301, "/")
	})

	rest.Post("/login", func(ctx *rex.Context) {
		user := ctx.FormValue("user").String()
		if user != "" {
			ctx.Session().Set("USER", user)
		}
		ctx.Redirect(301, "/")
	})

	rest.Get(
		"/logout",
		rex.Header("Cache-Control", "no-cache, no-store, must-revalidate"),
		func(ctx *rex.Context) {
			ctx.Session().Delete("USER")
			ctx.Redirect(301, "/")
		},
	)

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
