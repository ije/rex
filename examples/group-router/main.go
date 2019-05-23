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
	rest := rex.New()
	restUser := rex.New("user", "unused")
	restV2 := rex.New("v2")
	restV2User := rex.New("v2/user")
	restV3 := rex.New("v3")
	restV3User := rex.New("v3/user")

	rest.Get("/", func(ctx *rex.Context) {
		ctx.Html([]byte(indexHTML))
	})

	restUser.Get("/:id", func(ctx *rex.Context) {
		ctx.Ok("Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
	})

	restV2.Get("/", func(ctx *rex.Context) {
		ctx.Html([]byte(indexHTML2))
	})

	restV2User.Get("/:id", func(ctx *rex.Context) {
		ctx.Ok("[v2] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
	})

	restV3.Get("/", func(ctx *rex.Context) {
		ctx.Html([]byte(indexHTML3))
	})

	restV3User.Get("/:id", func(ctx *rex.Context) {
		ctx.Ok("[v3] Hello, I'm " + strings.Title(ctx.URL.Param("id")) + "!")
	})

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
