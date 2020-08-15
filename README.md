# REX

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)
[![GoReport](https://goreportcard.com/badge/github.com/ije/rex)](https://goreportcard.com/report/github.com/ije/rex)
[![MIT](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**REX** provides a query/mutation style API server in [Golang](https://golang.org/), noREST.


## Installing
```bash
go get github.com/ije/rex
```


## Usage

```go
package main

import (
    "sync"
    "github.com/ije/rex"
)

var posts sync.Map

func main() {
    // GET http://localhost/post?id=123
    rex.Query("post", func(ctx *rex.Context) interface{} {
        post, ok := posts.Load(ctx.Form.RequireInt("id"))
        if !ok {
            return rex.Error(404, "post not found")
        }
        return post
    })

    // POST http://localhost/add-post {"title": "Hello World"}
    rex.Mutation("add-post", func(ctx *rex.Context) interface{} {
        var id int
        posts.Range(func(k, v interface) {
            id++
        })
        post := map[string]interface{}{
            id: id + 1,
            title: ctx.Form.Value("title"),
        }
        posts.Store(id, post)
        return post
    })

    // POST http://localhost/remove-post {"id": 123}
    rex.Mutation("remove-post", func(ctx *rex.Context) interface{} {
        posts.Delete(ctx.Form.RequireInt("id"))
        return nil
    })

    rex.Start(8080)
}
```
