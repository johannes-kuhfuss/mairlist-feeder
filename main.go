package main

import (
	"log"

	"github.com/johannes-kuhfuss/mairlist-feeder/app"
)

func main() {
	if err := app.StartApp(); err != nil {
		log.Fatal(err)
	}
}
