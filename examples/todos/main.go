package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<h2>TODOS</h2>

{{if .user}}
    <p>Welcome back, <strong>{{.user}}</strong>!</p>

    <h3>Todos:</h3>
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

    <form method="post" action="/add-todo">
        <input name="todo" type="text">
        <input value="Add" type="submit">
    </form>

	<form method="post" action="/logout"> 
		<input value="Logout" type="submit">
	</form>

{{else}}
    <form method="post" action="/login">
        <input name="user" type="text">
		<input value="Login" type="submit">
    </form>
{{end}}
`

type user struct {
	name        string
	permissions []string
}

func (u *user) Permissions() []string {
	return u.permissions
}

func main() {
	todos := map[string][]string{}

	rex.Use(func(ctx *rex.Context) interface{} {
		if ctx.Session().Has("USER") {
			ctx.SetACLUser(&user{
				name:        string(ctx.Session().Get("USER")),
				permissions: []string{"add", "remove"},
			})
		}
		return rex.Next()
	})

	rex.Query("*", func(ctx *rex.Context) interface{} {
		data := map[string]interface{}{}
		aclUser := ctx.ACLUser()
		if aclUser != nil {
			data["user"] = aclUser.(*user).name
			data["todos"] = todos[aclUser.(*user).name]
		}
		return rex.RenderHTML(indexHTML, data)
	})

	rex.Mutation("add-todo", rex.ACL("add"), func(ctx *rex.Context) interface{} {
		todo := ctx.Form.Require("todo")
		user := ctx.ACLUser().(*user).name
		todos[user] = append(todos[user], todo)
		return rex.Redirect("/", 301)
	})

	rex.Mutation("delete-todo", rex.ACL("remove"), func(ctx *rex.Context) interface{} {
		index := ctx.Form.RequireInt("index")
		user := ctx.ACLUser().(*user).name
		_todos := todos[user]
		var newTodos []string
		for i, todo := range _todos {
			if i != int(index) {
				newTodos = append(newTodos, todo)
			}
		}
		todos[user] = newTodos
		return rex.Redirect("/", 301)
	})

	rex.Mutation("login", func(ctx *rex.Context) interface{} {
		user := ctx.Form.Value("user")
		if user != "" {
			ctx.Session().Set("USER", []byte(user))
		}
		return rex.Redirect("/", 301)
	})

	rex.Mutation("logout", func(ctx *rex.Context) interface{} {
		ctx.Session().Delete("USER")
		return rex.Redirect("/", 301)
	})

	rex.Use(rex.Debug())
	rex.Start(8080)
}
