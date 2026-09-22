package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/khorihutama/viewfinder-go/internal/android"
	"github.com/khorihutama/viewfinder-go/internal/native"
	"github.com/khorihutama/viewfinder-go/internal/renderer"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

type captureFrame struct {
	buffer []byte
	data   uint32
	width  uint32
	height uint32
	stride uint32
}

func readFrames(capture *native.Capture, buffers chan []byte, frames chan captureFrame, done <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)
	defer capture.Close()
	lastReadError := time.Time{}
	returnBuffer := func(buffer []byte) bool {
		select {
		case buffers <- buffer:
			return true
		case <-done:
			return false
		}
	}

	for {
		select {
		case <-done:
			return
		case buffer := <-buffers:
			if buffer == nil {
				return
			}
			data, width, height, stride, err := capture.ReadFrame(buffer)
			if err != nil {
				if lastReadError.IsZero() || time.Since(lastReadError) >= time.Second {
					log.Printf("ReadFrame failed: %v", err)
					lastReadError = time.Now()
				}
				if !returnBuffer(buffer) {
					return
				}
				select {
				case <-done:
					return
				case <-time.After(25 * time.Millisecond):
				}
				continue
			}
			lastReadError = time.Time{}

			if data == 0 {
				if !returnBuffer(buffer) {
					return
				}
				continue
			}

			select {
			case <-done:
				returnBuffer(buffer)
				return
			case frames <- captureFrame{buffer, data, width, height, stride}:
			default:
				select {
				case old := <-frames:
					returnBuffer(old.buffer)
				default:
				}
				select {
				case frames <- captureFrame{buffer, data, width, height, stride}:
				default:
					returnBuffer(buffer)
				}
			}
		}
	}
}

func main() {
	runtime.LockOSThread()

	log.Println("Starting Viewfinder Go")
	var androidDevice android.Device
	androidReady := false
	if devices, err := android.Devices(); err != nil {
		log.Printf("ADB unavailable: %v", err)
	} else {
		for _, device := range devices {
			log.Printf("Android device: %s (%s)", device.Serial, device.State)
		}
		if len(devices) == 0 {
			log.Println("No Android devices connected")
		} else if device, ok := android.FirstReady(devices); ok {
			if displayID, resolveErr := android.ResolveDisplayID(device); resolveErr == nil {
				device.Display = displayID
				log.Printf("Android external display: %d", displayID)
			} else {
				log.Printf("Android external display detection failed: %v", resolveErr)
				device.Display = -1
			}
			androidDevice = device
			androidReady = true
			if _, err := device.Shell("echo", "viewfinder-connected"); err != nil {
				log.Printf("ADB device command failed: %v", err)
			} else {
				log.Printf("ADB command ready: %s", device.Serial)
			}
		}
	}
	androidDone := make(chan struct{})
	androidChanges := make(chan []android.Device, 1)
	go android.Watch(androidDone, androidChanges)
	defer close(androidDone)

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

	bufferA := make([]byte, bufferSize)
	bufferB := make([]byte, bufferSize)
	buffers := make(chan []byte, 2)
	frames := make(chan captureFrame, 1)
	done := make(chan struct{})
	stopped := make(chan struct{})
	buffers <- bufferA
	buffers <- bufferB
	go readFrames(capture, buffers, frames, done, stopped)

	renderWidth := uint32(1280)
	renderHeight := uint32(720)
	var current *captureFrame
	fillMode := false
	fillKeyWasDown := false
	mouseWasDown := false
	lastTitle := time.Now()
	framesReceived := 0

	for running := true; running; {
		var pumpErr error
		running, pumpErr = window.Pump()
		if pumpErr != nil {
			log.Fatalf("window message loop failed: %v", pumpErr)
		}
		select {
		case devices := <-androidChanges:
			if device, ok := android.FirstReady(devices); ok {
				if displayID, resolveErr := android.ResolveDisplayID(device); resolveErr == nil {
					device.Display = displayID
					log.Printf("Android external display: %d", displayID)
				} else {
					device.Display = -1
					log.Printf("Android external display detection failed: %v", resolveErr)
				}
				androidDevice = device
				androidReady = true
				log.Printf("Android device ready: %s", device.Serial)
			} else {
				androidReady = false
				log.Println("Android device disconnected")
			}
		default:
		}
		fillKeyDown := window.KeyDown('F')
		if fillKeyDown && !fillKeyWasDown {
			fillMode = !fillMode
			if err := renderer.SetFillMode(fillMode); err != nil {
				log.Fatalf("renderer fill mode failed: %v", err)
			}
		}
		fillKeyWasDown = fillKeyDown
		mouseDown := window.KeyDown(win32.VK_LBUTTON)
		if mouseDown && !mouseWasDown && androidReady {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				if x >= 0 && y >= 0 && uint32(x) < clientWidth && uint32(y) < clientHeight {
					videoWidth := float64(selectedFormat.Width)
					videoHeight := float64(selectedFormat.Height)
					windowWidth := float64(clientWidth)
					windowHeight := float64(clientHeight)
					videoAspect := videoWidth / videoHeight
					windowAspect := windowWidth / windowHeight
					viewportWidth, viewportHeight := windowWidth, windowHeight
					if !fillMode {
						if videoAspect > windowAspect {
							viewportHeight = viewportWidth / videoAspect
						} else {
							viewportWidth = viewportHeight * videoAspect
						}
					}
					offsetX := (windowWidth - viewportWidth) / 2
					offsetY := (windowHeight - viewportHeight) / 2
					if float64(x) >= offsetX && float64(y) >= offsetY && float64(x) < offsetX+viewportWidth && float64(y) < offsetY+viewportHeight {
						deviceX := int((float64(x) - offsetX) * videoWidth / viewportWidth)
						deviceY := int((float64(y) - offsetY) * videoHeight / viewportHeight)
						log.Printf("Android tap: display=%d x=%d y=%d", androidDevice.Display, deviceX, deviceY)
						if err := androidDevice.Tap(deviceX, deviceY); err != nil {
							log.Printf("Android tap failed: %v", err)
						}
					}
				}
			}
		}
		mouseWasDown = mouseDown

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

		select {
		case frame := <-frames:
			if current != nil {
				buffers <- current.buffer
			}
			current = &frame
			framesReceived++
		default:
		}
		if time.Since(lastTitle) >= time.Second {
			if current != nil {
				fps := framesReceived / int(time.Since(lastTitle)/time.Second)
				if err := window.SetTitle(fmt.Sprintf("Viewfinder Go - %dx%d - %d FPS", current.width, current.height, fps)); err != nil {
					log.Printf("window title update failed: %v", err)
				}
			}
			framesReceived = 0
			lastTitle = time.Now()
		}

		if current != nil {
			if err := renderer.UploadNV12(current.buffer[:current.data], current.width, current.height, current.stride); err != nil {
				log.Fatalf("NV12 upload failed: %v", err)
			}
		}

		if current != nil {
			if err := renderer.Draw(); err != nil {
				log.Fatalf("renderer draw failed: %v", err)
			}
			if err := renderer.Present(); err != nil {
				log.Fatalf("renderer present failed: %v", err)
			}
		}
	}

	close(done)
	<-stopped
	if current != nil {
		buffers <- current.buffer
	}
}
