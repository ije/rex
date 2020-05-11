package main

import (
	"strings"

	"github.com/ije/rex"
)

const indexHTML = `
	<h1>Welcome to use REX!</h1>
	<p><a href="/users/bob">User Bob</a></p>
	<p><a href="/v2">V2 API</a></p>
	<p><a href="/v3">V3 API</a></p>
`
const v2HTML = `
	<h1>Welcome to use REX!</h1>
	<h2>V2 API</h2>
	<p><a href="/v2/users/bob">User Bob</a></p> 
	<p><a href="/">Home</a></p>
`
const v3HTML = `
	<h1>Welcome to use REX!</h1>
	<h2>V3 API</h2>
	<p><a href="/v3/users/bob">User Bob</a></p> 
	<p><a href="/">Home</a></p>
`

func main() {
	rex.Use(rex.Header("X-Version", "v1"), rex.Header("Foo", "bar"))
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})
	rex.Group("/users", func(r *rex.REST) {
		r.Get("/:id", func(ctx *rex.Context) {
			ctx.Ok("Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
		})
	})

	rex.Group("/v2", func(r *rex.REST) {
		r.Use(rex.Header("X-Version", "v2"))
		r.Get("/", func(ctx *rex.Context) {
			ctx.HTML(v2HTML)
		})
		r.Group("/users", func(r *rex.REST) {
			r.Get("/:id", func(ctx *rex.Context) {
				ctx.Ok("[v2] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
			})
		})
	})

	rex.Group("/v3", func(r *rex.REST) {
		r.Use(rex.Header("X-Version", "v3"))
		r.Get("/", func(ctx *rex.Context) {
			ctx.HTML(v3HTML)
		})
		r.Group("/users", func(r *rex.REST) {
			r.Get("/:id", func(ctx *rex.Context) {
				ctx.Ok("[v3] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
			})
		})
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}
