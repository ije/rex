WSX
====
**WSX** provides a simple & lightweight REST server by [golang](https://golang.org/) that can debug, build, and host a SPA(single page appliaction).

[![GoDoc](https://godoc.org/github.com/ije/wsx?status.svg)](https://godoc.org/github.com/ije/wsx)


Example
-------
```go
package main

import (
    "github.com/ije/wsx"
)

func main() {
    apis := &wsx.APIService{}
    apis.Get("/hello/:name", func(ctx *wsx.Context) {
        ctx.WriteText("Hello, " + ctx.URL.Params.ByName("name"))
    })

    wsx.Serve(&wsx.ServerConfig{
        AppRoot: "/var/www/app",
        Port: 8080,
    }, apis)
}
```
