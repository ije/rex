# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

Yet another web framework in Go.

## Installation

```bash
go get -u github.com/ije/rex
```

## Usage

```go
package main

import (
  "context"
  "log"
  "github.com/ije/rex"
)

func main() {
  // use middlewares
  rex.Use(
    rex.Logger(log.Default()),
    rex.Cors(rex.CorsAll()),
    rex.Compress(),
  )

  // match "GET /" route
  rex.GET("/{$}", func(ctx *rex.Context) any {
    return rex.Render(
      rex.Tpl("<h1>My Blog</h1><ul>{{range .}}<li>{{.Title}}</li>{{end}}</ul>"),
      posts.List(),
    )
  })

  // match "GET /posts/:id" route
  rex.GET("/posts/{id}", func(ctx *rex.Context) any {
    post, ok := posts.Get(ctx.PathValue("id"))
    if !ok {
      return rex.Err(404, "post not found")
    }
    return post
  })

  // match "POST /posts" route
  rex.POST("/posts", func(ctx *rex.Context) any {
    return posts.Add(ctx.FormValue("title"), ctx.FormValue("author"), ctx.FormValue("content"))
  })

  // match "DELETE /posts/:id" route
  rex.DELETE("/posts/{id}", func(ctx *rex.Context) any {
    ok := posts.Delete(ctx.PathValue("id"))
    return ok
  })

  // Starts the server
  <-rex.Start(context.Background(),80, nil)

  // Starts the server with autoTLS
  <-rex.StartWithAutoTLS(context.Background(), 443, nil)
}
```

More usages please check [examples/](./examples).

## Middleware

In **REX**, a middleware is a function that receives a `*rex.Context` and returns a `any`. If the returned value is not `rex.Next()`, the middleware will return the value to the client, or continue to execute the next middleware.

```go
rex.Use(func(ctx *rex.Context) any {
  if ctx.Pathname() == "/hello" {
    // return a html response
    return rex.HTML("<h1>hello world</h1>")
  }

  // use next handler
  return ctx.Next()
})
```

## Routing

**REX** uses [ServeMux Patterns](https://pkg.go.dev/net/http#ServeMux) (requires Go 1.22+) to define routes.

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

You can access the path params via calling `ctx.PathValue(paramName)`:

```go
rex.GET("/posts/{id}", func(ctx *rex.Context) any {
  return fmt.Sprintf("ID is %s", ctx.PathValue("id"))
})
```
