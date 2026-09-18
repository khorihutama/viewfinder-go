package main

import (
	"log"

	"github.com/khorihutama/viewfinder-go/internal/native"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

const (
	windowWidth  = 1280
	windowHeight = 720
)

func main() {
	window, err := win32.Create(
		"Viewfinder-Go",
		windowWidth,
		windowHeight,
	)
	if err != nil {
		log.Fatalf(
			"create window: %v",
			err,
		)
	}

	log.Println(
		"Win32 window created",
	)

	log.Println(
		"Creating D3D11 renderer...",
	)

	renderer, err := native.CreateRenderer(
		window.HWND,
		windowWidth,
		windowHeight,
	)
	if err != nil {
		log.Fatalf(
			"create renderer: %v",
			err,
		)
	}

	defer renderer.Close()

	log.Println(
		"D3D11 renderer initialized",
	)

	if err := renderer.Clear(
		0.05,
		0.05,
		0.08,
		1.0,
	); err != nil {
		log.Fatalf(
			"clear renderer: %v",
			err,
		)
	}

	if err := renderer.Present(); err != nil {
		log.Fatalf(
			"present renderer: %v",
			err,
		)
	}

	log.Println(
		"Frame presented",
	)

	if err := window.Run(); err != nil {
		log.Fatalf(
			"message loop: %v",
			err,
		)
	}
}
