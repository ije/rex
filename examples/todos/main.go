package main

import (
	"github.com/ije/rex"
)

const indexHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>TODOS</title>
</head>
<body>
    <h1>TODOS</h1>

    {{if .user}}
        <form method="post" action="/logout"> 
            <p>Welcome back, <strong>{{.user}}</strong>!</p>
            <p><input value="Logout" type="submit"></p>
        </form>
        <h2>Todos List:</h2>
        <ul>
            {{range $index,$todo := .todos}}
            <li>
                <form style="display:inline-block;" method="post" action="/delete-todo">
                    {{$todo}} &nbsp; <input name="index" type="hidden" value="{{$index}}"> <input value="X" type="submit">
                </form>
            </li>
            {{end}}
        </ul>
        <form method="post" action="/add-todo">
            <input name="todo" type="text">
            <input value="Add" type="submit">
        </form>
    {{else}}
        <form method="post" action="/login">
            <input name="user" type="text">
            <input value="Login" type="submit">
        </form>
    {{end}}
</body>
</html>
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
		return nil
	})

	rex.Query("*", func(ctx *rex.Context) interface{} {
		data := map[string]interface{}{}
		aclUser := ctx.ACLUser()
		if aclUser != nil {
			data["user"] = aclUser.(*user).name
			data["todos"] = todos[aclUser.(*user).name]
		}
		return rex.HTML(indexHTML, data)
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
		userTodos := todos[user]
		var newTodos []string
		for i, todo := range userTodos {
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

	rex.Start(8080)
}
