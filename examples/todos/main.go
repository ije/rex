package main

import (
	"fmt"
	"strconv"

	"github.com/ije/rex"
)

const indexHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Todos by REX</title>
</head>
<body>
    <h1>TODOS</h1>
    {{if .user}}
        <form method="post" action="/logout">
            <p>Welcome back, <strong>{{.user}}</strong>! <input value="Logout" type="submit"></p>
        </form>
        <h2>Todos List:</h2>
        <ul>
            {{range $index,$todo := .todos}}
            <li>
                <form style="display:inline-block;" method="post" action="/delete-todo">
                    {{$todo}} &nbsp; <input name="_method" type="hidden" value="DELETE"> <input name="index" type="hidden" value="{{$index}}"> <input value="x" type="submit">
                </form>
            </li>
            {{end}}
        </ul>
		<form method="post" action="/add-todo">
			<input name="todo" type="text" placeholder="Add">
		</form>
    {{else}}
        <form method="post" action="/login">
            <input name="user" type="text">
            <input value="Login" type="submit">
        </form>
				<p>Try to login with <strong onclick="document.querySelector('input[name=user]').value='admin';" style="cursor:pointer;">admin</strong> or <strong onclick="document.querySelector('input[name=user]').value='guest';" style="cursor:pointer;">guest</strong>.</p>
    {{end}}
</body>
</html>
`

var indexTpl = rex.Tpl("html", indexHTML)

type user struct {
	name  string
	perms []string
}

func (u *user) Perms() []string {
	return u.perms
}

func main() {
	todos := []string{}

	// override http methods middleware
	rex.Use(func(ctx *rex.Context) interface{} {
		if ctx.R.Method == "POST" && ctx.FormValue("_method") == "DELETE" {
			ctx.R.Method = "DELETE"
		}
		return nil
	})

	// auth middleware
	rex.Use(rex.AclAuth(func(ctx *rex.Context) rex.AclUser {
		value := ctx.Session().Get("USER")
		if value == nil {
			return nil
		}
		name := string(value)
		if name == "admin" {
			return &user{
				name:  "admin",
				perms: []string{"add", "remove"},
			}
		} else if name == "guest" {
			return &user{
				name: name,
			}
		}
		return nil
	}))

	rex.GET("/{$}", func(ctx *rex.Context) interface{} {
		data := map[string]interface{}{}
		aclUser := ctx.AclUser()
		if aclUser != nil {
			data["user"] = aclUser.(*user).name
			data["todos"] = todos
		}
		return rex.Render(indexTpl, data)
	})

	rex.POST("/add-todo", rex.Perm("add"), func(ctx *rex.Context) interface{} {
		todo := ctx.FormValue("todo")
		todos = append(todos, todo)
		return rex.Redirect("/", 302)
	})

	rex.DELETE("/delete-todo", rex.Perm("remove"), func(ctx *rex.Context) interface{} {
		index, err := strconv.ParseInt(ctx.FormValue("index"), 10, 64)
		if err != nil {
			return err
		}
		var newTodos []string
		for i, todo := range todos {
			if i != int(index) {
				newTodos = append(newTodos, todo)
			}
		}
		todos = newTodos
		return rex.Redirect("/", 302)
	})

	rex.POST("/login", func(ctx *rex.Context) interface{} {
		user := ctx.FormValue("user")
		if user != "admin" && user != "guest" {
			return rex.Status(403, rex.HTML("<p>Oops, you are not allowed to login. <a href=\"/\">Go back</a></p>"))
		}
		ctx.Session().Set("USER", []byte(user))
		return rex.Redirect("/", 302)
	})

	rex.POST("/logout", func(ctx *rex.Context) interface{} {
		ctx.Session().Delete("USER")
		return rex.Redirect("/", 302)
	})

	fmt.Println("Server running on http://localhost:8080")
	<-rex.Start(8080)
}
