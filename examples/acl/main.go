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
	id         interface{}
	privileges []string
}

func (u *User) ID() interface{} {
	return u.id
}

func (u *User) Privileges() []string {
	return u.privileges
}

var todos = map[string][]string{}

func main() {
	rest := rex.New()
	rest.Template = template.Must(template.New("index").Parse(indexHTML))

	rest.Use(
		rex.ACL(func(id interface{}) acl.User {
			return &User{
				id:         id,
				privileges: []string{"add"},
			}
		}),
	)

	rest.Get("/", func(ctx *rex.Context) {
		if user := ctx.User(); user != nil {
			ctx.Render("index", map[string]interface{}{
				"user":  user.ID(),
				"todos": todos[user.ID().(string)],
			})
		} else {
			ctx.Render("index", nil)
		}
	})

	rest.Post("/add-todo", rex.Privileges("add"), func(ctx *rex.Context) {
		todo := ctx.FormString("todo", "")
		if todo != "" {
			user := ctx.User().ID().(string)
			todos[user] = append(todos[user], todo)
		}
		ctx.Redirect(301, "/")
	})

	rest.Post("/login", func(ctx *rex.Context) {
		user := ctx.FormString("user", "")
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
