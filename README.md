WEBX
====

webx provides a API server to process data. With nodejs, webx can debug,build&host a SPA(single page appliaction).
[![GoDoc](https://godoc.org/github.com/ije/webx?status.svg)](https://godoc.org/github.com/ije/webx)
 
 
Example
-------

```go
package main

import (
	"github.com/ije/webx"
	"github.com/ije/webx/user"
)

func main() {
	var apis = web.APIService{}

	apis.Get("hello", func(ctx *webx.Context) {
		ctx.JSON(200, map[string]string{
			"words": "Hello, " + ctx.FormString("name"),
		})
	}, user.Privilege_Open)

	webx.Register(apis)

	webx.Serve("/var/www/spa-app", nil)
}
```
 
 
Features
--------

* Restful API Server
* SPA Templates (React,Angular,Vue,ect...)
 
 
Node.js
-------

Please install the latest nodejs if you are using a SPA template with **webpack**.
