package main

import (
	"strings"

	"github.com/ije/rex"
)

const (
	indexHTML = `
<h1>Welcome to use REX!</h1>
<p><a href="/user/bob">User Bob</a></p>
<p><a href="/v2">V2 API</a></p>
<p><a href="/v3">V3 API</a></p> 
<p><a href="/default">*default</a></p> 
`
	indexHTML2 = `
<h1>V2 API</h1>
<p><a href="/v2/user/bob">User Bob</a></p> 
<p><a href="/">Home</a></p>
`
	indexHTML3 = `
<h1>V3 API</h1>
<p><a href="/v3/user/bob">User Bob</a></p> 
<p><a href="/">Home</a></p>
`
)

func main() {
	rex.Use(rex.Header("X-Version", "default"), rex.Header("Foo", "bar"))
	rex.Get("/default", func(ctx *rex.Context) {
		ctx.Ok("default")
	})

	rest := rex.New()
	rest.Use(rex.Header("X-Version", "v1"), rex.Header("Title", "Welcome to use REX!"))
	rest.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})
	rest.Group("/user", func(r *rex.REST) {
		r.Get("/:id", func(ctx *rex.Context) {
			ctx.Ok("Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
		})
	})

	restV2 := rex.New().Prefix("v2")
	restV2.Use(rex.Header("X-Version", "v2"))
	restV2.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML2)
	})
	restV2.Group("/user", func(r *rex.REST) {
		r.Get("/:id", func(ctx *rex.Context) {
			ctx.Ok("[v2] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
		})
	})

	restV3 := rex.New().Prefix("v3")
	restV3.Use(rex.Header("X-Version", "v3"))
	restV3.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML3)
	})
	restV3.Group("/user", func(r *rex.REST) {
		r.Get("/:id", func(ctx *rex.Context) {
			ctx.Ok("[v3] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
		})
	})

	rex.Start(8080)
}
