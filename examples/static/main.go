package main

import (
	"github.com/ije/rex"
)

func main() {
	rex.Use(rex.Header("Server", "nginx"))

	rex.Get("/*path", rex.Static("./www", "e404.html"))

	rex.Start(8080)
}
