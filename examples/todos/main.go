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
	{{range $index,$todo := .todos}}
	<li>
		{{$todo}} &nbsp;
		<form style="display:inline-block;" method="post" action="/delete-todo">
			<input name="index" type="hidden" value="{{$index}}">
			<input value="X" type="submit">
		</form>
	</li>
	{{end}}
</ul>
<div>

<form method="post" action="/add-todo">
	<input name="todo" type="text">
	<input value="Add" type="submit">
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
				permissions: []string{"add", "remove"},
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
		todo := ctx.Form.Require("todo")
		user := ctx.ACLUser().(*user).id
		todos[user] = append(todos[user], todo)
		ctx.Redirect("/", 301)
	})

	rex.Post("/delete-todo", rex.ACL("remove"), func(ctx *rex.Context) {
		index := ctx.Form.RequireInt("index")
		user := ctx.ACLUser().(*user).id
		_todos := todos[user]
		var newTodos []string
		for i, todo := range _todos {
			if i != int(index) {
				newTodos = append(newTodos, todo)
			}
		}
		todos[user] = newTodos
		ctx.Redirect("/", 301)
	})

	rex.Post("/login", func(ctx *rex.Context) {
		user := ctx.Form.Get("user")
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

	rex.Use(rex.SendError())
	rex.Start(8080)
}
