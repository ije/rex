REX
====
**REX** provides a simple & light-weight REST server in [Golang](https://golang.org/) that can debug, build, and host a SPA(single page appliaction).

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)


Example
-------
```go
package main

import (
    "github.com/ije/rex"
)

func main() {
    apis := &rex.APIService{}

    apis.Get("/hello/:name", func(ctx *rex.Context) {
        ctx.WriteString("Hello, " + ctx.URL.Params.ByName("name"))
    })

    rex.Serve(&rex.ServerConfig{
        AppRoot: "/var/www/app",
        Port: 8080,
    })
}
```
