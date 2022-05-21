package main

import (
	"fmt"

	"github.com/ije/rex"
)

func main() {
	rex.Use(rex.Compression())

	rex.Query("*", func(ctx *rex.Context) interface{} {
		return rex.FS("./www", "e404.html")
	})

	fmt.Println("Server running on http://localhost:8080")
	<-rex.Start(8080)
}
