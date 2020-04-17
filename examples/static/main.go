package main

import (
	"github.com/ije/rex"
)

func main() {
	rex.Get("/*", rex.Static("./www", "e404.html"))

	rex.Start(8080)
}
