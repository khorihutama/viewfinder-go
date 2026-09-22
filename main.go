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

	if err := renderer.InitializeWindow(
		window.HWND,
		1280,
		720,
	); err != nil {
		log.Fatalf(
			"failed to initialize renderer window: %v",
			err,
		)
	}

	log.Println("D3D11 swap chain initialized")

	defer renderer.DestroyWindowRenderer()

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
	renderWidth := uint32(1280)
	renderHeight := uint32(720)

	for running := true; running; {
		var pumpErr error
		running, pumpErr = window.Pump()
		if pumpErr != nil {
			log.Fatalf("window message loop failed: %v", pumpErr)
		}

		clientWidth, clientHeight, err := window.ClientSize()
		if err != nil {
			log.Fatalf("failed to read window size: %v", err)
		}

		if clientWidth != 0 && clientHeight != 0 &&
			(clientWidth != renderWidth || clientHeight != renderHeight) {
			if err := renderer.Resize(clientWidth, clientHeight); err != nil {
				log.Fatalf("renderer resize failed: %v", err)
			}

			renderWidth = clientWidth
			renderHeight = clientHeight
		}

		dataSize,
			width,
			height,
			stride,
			err := capture.ReadFrame(
			buffer,
		)

		if err != nil {
			log.Printf("ReadFrame failed: %v", err)

			continue
		}

		if dataSize == 0 {
			continue
		}

		err = renderer.UploadNV12(
			buffer[:dataSize],
			width,
			height,
			stride,
		)

		if err != nil {
			log.Fatalf(
				"NV12 upload failed: %v",
				err,
			)
		}

		if err := renderer.Draw(); err != nil {
			log.Fatalf("renderer draw failed: %v", err)
		}

		if err := renderer.Present(); err != nil {
			log.Fatalf("renderer present failed: %v", err)
		}
	}
}
