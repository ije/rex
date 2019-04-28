package main

import (
	"github.com/ije/rex"
)

func main() {
	rest := rex.New()
	rest.Use(rex.Header("Server", "nginx"))

	rest.Get("/*path", rex.Static("./root", "e404.html"))

	rex.Serve(rex.Config{
		Port:  8080,
		Debug: true,
	})
}
