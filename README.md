# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**REX** provides a query/mutation style API server in [Golang](https://golang.org/), noREST.

## Installing

```bash
go get -u github.com/ije/rex
```

## Example

```go
package main

import (
    "github.com/ije/rex"
)

func main() {
    // GET /*
    rex.Query("*", func(ctx *rex.Context) interface{} {
        return rex.HTML(
            200,
            "<h1>My Blog</h1>{{range .}}{{.Title}}{{end}}",
            blogs.All(),
        )
    })

    // GET /post/123 => Blog JSON
    rex.Query("blog", func(ctx *rex.Context) interface{} {
        blog, ok := blogs.Load(ctx.URL.RequireIntSegment(1))
        if !ok {
            return &rex.Error{404, "blog not found"}
        }
        return blog
    })

    // POST /add-blog {"title": "Hello World"} => Blog JSON
    rex.Mutation("add-blog", func(ctx *rex.Context) interface{} {
        blog := NewBlog(ctx.Form.Value("title"))
        blogs.Store(blog)
        return blog
    })

    rex.Start(8080)
}
```
