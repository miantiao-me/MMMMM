package main

import (
	"log"

	"github.com/miantiao-me/MMMMM/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
