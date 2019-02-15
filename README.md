WSX
====
**WSX** provides a restful API server by [golang](https://golang.org/) that can debug, build, and host a SPA(single page appliaction).

[![GoDoc](https://godoc.org/github.com/ije/wsx?status.svg)](https://godoc.org/github.com/ije/wsx)


Example
-------
```go
package main

import (
	"github.com/ije/wsx"
)

func main() {
	var apis = wsx.NewAPIService()

	apis.Get("/hello/:name", func(ctx *wsx.Context) {
		ctx.WriteJSON(200, map[string]string{
			"message": "Hello, " + ctx.URL.Params.ByName("name"),
		})
	})

	wsx.Serve(&wsx.ServerConfig{
		AppRoot: "/var/www/app",
		Port: 8080,
	}, apis)
}
```
