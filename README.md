# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**REX** is a lightweight, high-performance, and middleware-extensible web framework in Go. Used by [esm.sh](https://esm.sh) project.

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

  rex.GET("/{$}", func(ctx *rex.Context) interface{} {
    return rex.Render(
      rex.Tpl("html", "<h1>My Blog</h1><ul>{{range .}}<li>{{.Title}}</li>{{end}}</ul>"),
      posts.List(),
    )
  })

  rex.GET("/posts/{id}", func(ctx *rex.Context) interface{} {
    post, ok := posts.Get(ctx.PathValue("id"))
    if !ok {
      return &rex.Error{404, "post not found"}
    }
    return post
  })

  rex.POST("/posts", func(ctx *rex.Context) interface{} {
    return posts.Add(ctx.FormValue("title"), ctx.FormValue("author"), ctx.FormValue("content"))
  })

  rex.DELETE("/posts/{id}", func(ctx *rex.Context) interface{} {
    ok := posts.Delete(ctx.PathValue("id"))
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

**REX** uses [ServeMux Patterns](https://pkg.go.dev/net/http#hdr-Patterns) to match the request route.

Patterns can match the method, host and path of a request. Some examples:

- `/index.html` matches the path `/index.html` for any host and method.
- `GET /static/` matches a GET request whose path begins with `/static/`.
- `example.com/` matches any request to the host `example.com`.
- `example.com/{$}` matches requests with host `example.com` and path `/`.
- `/b/{bucket}/o/{objectname...}` matches paths whose first segment is `b` and whose third segment is `o`. The name `bucket` denotes the second segment and `objectname` denotes the remainder of the path.

In general, a pattern looks like:
```
[METHOD ][HOST]/[PATH]
```

you can access the path params via the `ctx.PathValue` method:

```go
rex.GET("/posts/{id}", func(ctx *rex.Context) interface{} {
  return fmt.Sprintf("id is %s", ctx.PathValue("id"))
})
```
