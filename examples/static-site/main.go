package main

import (
	"fmt"

	"github.com/ije/rex"
)

func main() {
	rex.Use(
		rex.Compress(),
		rex.Static("./www", "e404.html"),
	)

	fmt.Println("Server running on http://localhost:8080")
	<-rex.Start(8080)
}
