package main

import (
	"log"

	"github.com/khorihutama/viewfinder-go/internal/renderer"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

func main() {
	window, err := win32.Create(
		"Viewfinder-Go",
		1280,
		720,
	)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}

	log.Println("Win32 window created")

	device, err := renderer.Initialize()
	if err != nil {
		log.Fatalf("initialize D3D11: %v", err)
	}

	log.Printf(
		"D3D11 device initialized: pointer=0x%X feature_level=0x%X",
		device.DevicePointer(),
		device.FeatureLevel(),
	)

	log.Printf(
		"D3D11 context initialized: pointer=0x%X",
		device.ContextPointer(),
	)

	if err := window.Run(); err != nil {
		log.Fatalf("message loop: %v", err)
	}
}
