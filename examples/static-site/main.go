package main

import (
	"context"
	"fmt"

	"github.com/ije/rex"
)

func main() {
	rex.Use(
		rex.Compress(),
		rex.Static("./www", "e404.html"),
	)

	<-rex.Start(context.Background(), 8080, func(port uint16) {
		fmt.Printf("Server running on http://localhost:%d\n", port)
	})
}
