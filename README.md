# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**REX** is a lightweight, high-performance, and middleware-extensible web framework in Go. Used by [esm.sh](https://esm.sh) CDN.

## Installing

```bash
go get -u github.com/ije/rex
```

## Example

```go
package main

import (
  "log"

  "github.com/ije/rex"
)

func main() {
  // use middlewares
  rex.Use(
    rex.Logger(log.Default()),
    rex.Cors(rex.CorsAllowAll()),
    rex.Compress(),
  )

  // GET / => Post list in HTML
  rex.GET("/", func(ctx *rex.Context) interface{} {
    return rex.Render(
      rex.Tpl("html", "<h1>My Blog</h1><ul>{{range .}}<li>{{.Title}}</li>{{end}}</ul>"),
      posts.GetAll(),
    )
  })

  // GET /posts/foo-bar => Post in JSON if exists
  rex.GET("/posts/:slug", func(ctx *rex.Context) interface{} {
    post, ok := posts.Get(ctx.Path.Params.Get("slug"))
    if !ok {
      return &rex.Error{404, "post not found"}
    }
    return post
  })

  // POST /posts {"title": "Hello World"} => Created Post in JSON
  rex.POST("/posts", func(ctx *rex.Context) interface{} {
    post := Newpost(ctx.Form.Value("title"))
    posts.Create(post)
    return post
  })

  // DELETE /posts/foo-bar => "true" if deleted
  rex.DELETE("/posts/:slug", func(ctx *rex.Context) interface{} {
    ok := posts.Delete(ctx.Path.Params.Get("slug"))
    return ok
  })

  // Starts the server
  <-rex.Start(80)

  // Starts the server with TLS (powered by Let's Encrypt)
  <-rex.StartWithAutoTLS(443)
}
```

More examples check [examples](./examples).

## Middleware

In **REX**, a middleware is a function that receives a `*rex.Context` and returns a `interface{}`. If the returned value is not `nil`, the middleware will return the value to the client, or continue to execute the next middleware.

```go
rex.Use(func(ctx *rex.Context) interface{} {
  // return a html response
  return rex.HTML("<h1>hello world</h1>")

  // return nil to continue next handler
  return nil
})
```

## Router

**REX** uses [httprouter](https://github.com/julienschmidt/httprouter) as the router, so you can use the same syntax as httprouter to define routes.

```go
// static route
rex.GET("/", func(ctx *rex.Context) interface{} {})
// dynamic route
rex.GET("/posts/:slug", func(ctx *rex.Context) interface{} {})
// match all
rex.GET("/posts/*path", func(ctx *rex.Context) interface{} {})
```

you can access the path params via `ctx.Path.Params`:

```go
rex.GET("/posts/:slug", func(ctx *rex.Context) interface{} {
  return fmt.Sprintf("slug is %s", ctx.Path.Params.Get("slug"))
})
```
