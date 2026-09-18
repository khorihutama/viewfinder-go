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
	log.Println("Capture devices:")

	for _, device := range devices {
		log.Printf(
			"[%d] %s",
			device.Index,
			device.Name,
		)

		formats, err := capture.Formats(
			device.Index,
		)

		if err != nil {
			log.Printf(
				"    failed to enumerate formats: %v",
				err,
			)

			continue
		}

		for _, format := range formats {
			fps := float64(
				format.FPSNumerator,
			) / float64(
				format.FPSDenominator,
			)

			log.Printf(
				"    [%d] %dx%d @ %.2f FPS | %s",
				format.Index,
				format.Width,
				format.Height,
				fps,
				format.Subtype,
			)
		}
	}

	log.Println("")
	log.Println(
		"Press Ctrl+C or close the window to exit.",
	)

	if err := window.Run(); err != nil {
		log.Fatalf(
			"message loop: %v",
			err,
		)
	}

	log.Println("Window closed")
}
