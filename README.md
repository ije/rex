# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**REX** is a lightweight, high-performance, and extensible web framework for Go.

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

  // GET /*
  rex.GET("/*", func(ctx *rex.Context) interface{} {
    return rex.HTML(
      "<h1>My Blog</h1><ul>{{range .}}<li>{{.Title}}</li>{{end}}</ul>",
      blogs.All(),
    )
  })

  // GET /post/123 => Blog JSON
  rex.GET("/post/?", func(ctx *rex.Context) interface{} {
    blog, ok := blogs.Get(ctx.Path.RequireSegment(2))
    if !ok {
      return &rex.Error{404, "blog not found"}
    }
    return blog
  })

  // POST /add-blog {"title": "Hello World"} => Blog JSON
  rex.POST("/create-blog", func(ctx *rex.Context) interface{} {
    blog := NewBlog(ctx.Form.Value("title"))
    blogs.Create(blog)
    return blog
  })

    // DELETE /add-blog?id=123 => Boolean
  rex.DELETE("/delete-blog", func(ctx *rex.Context) interface{} {
    ok := blogs.Delete(ctx.Form.RequireInt("id"))
    return ok
  })

  // Starts the server
  rex.Start(8080)
}
```
