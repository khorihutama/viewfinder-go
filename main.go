package main

import (
	"log"

	"github.com/khorihutama/viewfinder-go/internal/win32"
)

func main() {
	window, err := win32.Create(
		"Viewfinder-Go",
		1280, 720,
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Window created")

	if err := window.Run(); err != nil {
		log.Fatal(err)
	}

	log.Println("Application exited")

}
