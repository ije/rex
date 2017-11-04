WEBX
====
webx provides a restful API server by [golang](https://golang.org/) that can debug, build and host a SPA(single page appliaction).

[![GoDoc](https://godoc.org/github.com/ije/webx?status.svg)](https://godoc.org/github.com/ije/webx)


Example
-------
```go
package main

import (
	"github.com/ije/webx"
)

func main() {
	var apis = &web.APIService{}

	apis.Get("/hello/:name", func(ctx *webx.Context, xs *webx.XService) {
		ctx.WriteJSON(200, map[string]string{
			"message": "Hello, " + ctx.URL.Params.ByName("name"),
		})
	}, "privilegeId")

	webx.Register(apis)

	webx.Serve(&webx.ServerConfig{
		AppRoot: "/var/www/spa-app",
		Port: 8080,
	})
}
```


Features
--------
* Restful API Server
* Debug and Build SPA


Node.js
-------
If you run the webx with a SPA, you need install the [nodejs](https://nodejs.org/) to debug and build the App.
