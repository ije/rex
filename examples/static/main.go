package main

import (
	"github.com/ije/rex"
)

func main() {
	rex.Get("/*filepath", rex.Static("./www", "e404.html"))

	rex.Start(8080)
}
