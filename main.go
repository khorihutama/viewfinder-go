package main

import (
	"log"
	"runtime"

	"github.com/khorihutama/viewfinder-go/internal/native"
	"github.com/khorihutama/viewfinder-go/internal/renderer"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

func main() {
	runtime.LockOSThread()

	log.Println("Starting Viewfinder Go")

	// ------------------------------------------------------------
	// Create Win32 window
	// ------------------------------------------------------------

	window, err := win32.Create(
		"Viewfinder Go",
		1280,
		720,
	)

	if err != nil {
		log.Fatalf(
			"failed to create window: %v",
			err,
		)
	}

	log.Println("Win32 window created")

	// ------------------------------------------------------------
	// Initialize D3D11
	// ------------------------------------------------------------

	d3d, err := renderer.Initialize()

	if err != nil {
		log.Fatalf(
			"failed to initialize D3D11: %v",
			err,
		)
	}

	log.Printf(
		"D3D11 device initialized: pointer=0x%X feature_level=0x%X",
		d3d.DevicePointer(),
		d3d.FeatureLevel(),
	)

	log.Printf(
		"D3D11 context initialized: pointer=0x%X",
		d3d.ContextPointer(),
	)

	// ------------------------------------------------------------
	// Initialize capture
	// ------------------------------------------------------------

	capture, err := native.CreateCapture()

	if err != nil {
		log.Fatalf(
			"failed to create capture: %v",
			err,
		)
	}

	defer capture.Destroy()

	log.Println("Capture object created")

	if err := capture.Initialize(); err != nil {
		log.Fatalf(
			"failed to initialize capture: %v",
			err,
		)
	}

	log.Println("Capture initialized")

	// ------------------------------------------------------------
	// Enumerate devices
	// ------------------------------------------------------------

	devices, err := capture.Devices()

	if err != nil {
		log.Fatalf(
			"failed to enumerate devices: %v",
			err,
		)
	}

	log.Printf(
		"Found %d capture device(s)",
		len(devices),
	)

	if len(devices) == 0 {
		log.Fatal("no capture devices found")
	}

	for _, device := range devices {
		log.Printf(
			"Device [%d]: %s",
			device.Index,
			device.Name,
		)
	}

	// ------------------------------------------------------------
	// Select first capture device
	// ------------------------------------------------------------

	deviceIndex := devices[0].Index

	log.Printf(
		"Using device [%d]: %s",
		deviceIndex,
		devices[0].Name,
	)

	// ------------------------------------------------------------
	// Enumerate formats
	// ------------------------------------------------------------

	formats, err := capture.Formats(
		deviceIndex,
	)

	if err != nil {
		log.Fatalf(
			"failed to enumerate formats: %v",
			err,
		)
	}

	log.Printf(
		"Found %d format(s)",
		len(formats),
	)

	for _, format := range formats {
		fps := float64(
			format.FPSNumerator,
		) / float64(
			format.FPSDenominator,
		)

		log.Printf(
			"[%d] %dx%d @ %.2f FPS | %s",
			format.Index,
			format.Width,
			format.Height,
			fps,
			format.Subtype,
		)
	}

	if len(formats) == 0 {
		log.Fatal("no capture formats found")
	}

	// ------------------------------------------------------------
	// Select first format
	// ------------------------------------------------------------

	formatIndex := formats[0].Index
	selectedFormat := formats[0]

	log.Printf(
		"Opening format [%d]: %dx%d",
		formatIndex,
		selectedFormat.Width,
		selectedFormat.Height,
	)

	if err := capture.Open(
		deviceIndex,
		formatIndex,
	); err != nil {
		log.Fatalf(
			"failed to open capture stream: %v",
			err,
		)
	}

	log.Println("Capture stream opened")

	if !capture.IsOpen() {
		log.Fatal("capture stream reports as closed")
	}

	log.Println("Capture stream is open")

	// ------------------------------------------------------------
	// Read video frames
	// ------------------------------------------------------------

	/*
	 * Allocate a buffer large enough for common
	 * uncompressed formats.

	 * We will improve this later based on the
	 * actual media subtype.
	 */

	bufferSize :=
		int(selectedFormat.Width) *
			int(selectedFormat.Height) *
			4

	buffer := make([]byte, bufferSize)

	for i := 0; i < 10; i++ {
		dataSize,
			width,
			height,
			stride,
			err := capture.ReadFrame(
			buffer,
		)

		if err != nil {
			log.Printf(
				"ReadFrame %d failed: %v",
				i,
				err,
			)

			continue
		}

		log.Printf(
			"Frame %d: %dx%d | data=%d bytes | stride=%d",
			i,
			width,
			height,
			dataSize,
			stride,
		)
	}

	log.Println("Frame test completed")

	// ------------------------------------------------------------
	// Keep window alive
	// ------------------------------------------------------------

	if err := window.Run(); err != nil {
		log.Fatalf(
			"window message loop failed: %v",
			err,
		)
	}
}
