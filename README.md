# REX

**REX** provides a simple & light-weight REST server in [Golang](https://golang.org/).

[![GoDoc](https://godoc.org/github.com/ije/rex?status.svg)](https://godoc.org/github.com/ije/rex)


### Example
```go
package main

import (
	"github.com/ije/rex"
)

func main() {
	rex.Get("/hello/:name", func(ctx *rex.Context) {
		ctx.Ok("Hello, " + ctx.URL.Param("name"))
	})
	
	rex.Start(8080)
}
```
