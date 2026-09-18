package main

import (
	"log"
	"runtime"

	"github.com/khorihutama/viewfinder-go/internal/native"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

const (
	windowWidth  = 1280
	windowHeight = 720
)

func main() {
	runtime.LockOSThread()

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

	log.Println("D3D11 renderer initialized")

	capture, err := native.CreateCapture()
	if err != nil {
		log.Fatalf(
			"create capture: %v",
			err,
		)
	}

	defer capture.Close()

	log.Println(
		"Media Foundation capture object created",
	)

	if err := capture.Initialize(); err != nil {
		log.Fatalf(
			"initialize capture: %v",
			err,
		)
	}

	log.Println(
		"Media Foundation initialized",
	)

	devices, err := capture.Devices()
	if err != nil {
		log.Fatalf(
			"enumerate capture devices: %v",
			err,
		)
	}

	log.Println("")

	if len(devices) == 0 {
		log.Println(
			"No video capture devices found",
		)
	} else {
		log.Printf(
			"Found %d video capture device(s):",
			len(devices),
		)

		for _, device := range devices {
			log.Printf(
				"[%d] %s",
				device.Index,
				device.Name,
			)
		}
	}

	log.Println("")
	log.Println(
		"Press Ctrl+C or close the window to exit.",
	)

	// Keep the Win32 window responsive.
	if err := window.Run(); err != nil {
		log.Fatalf(
			"message loop: %v",
			err,
		)
	}

	log.Println("Window closed")
}
