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

	log.Println("Win32 window created")

	log.Println(
		"Creating D3D11 device and DXGI swap chain...",
	)

	swapChain, err := native.CreateSwapChain(
		window.HWND,
		windowWidth,
		windowHeight,
	)
	if err != nil {
		log.Fatalf(
			"create swap chain: %v",
			err,
		)
	}

	log.Printf(
		"DXGI swap chain initialized: 0x%X",
		swapChain,
	)

	if err := window.Run(); err != nil {
		log.Fatalf(
			"message loop: %v",
			err,
		)
	}
}
