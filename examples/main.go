package main

import (
	"log"

	kite "github.com/claudiovictors/kite/core"
)

func main() {
	app := kite.New()

	app.Get("/", func(req kite.Request, res kite.Response) error {
		
		return res.Send("Hello, World")
	})

	log.Fatal(app.Listen(":3000"))
}
