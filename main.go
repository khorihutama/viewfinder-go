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

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type captureFrame struct {
	buffer []byte
	data   uint32
	width  uint32
	height uint32
	stride uint32
}

func mapClientToVideo(x, y int32, clientWidth, clientHeight, videoWidth, videoHeight uint32, fill bool) (int, int, bool) {
	if x < 0 || y < 0 || uint32(x) >= clientWidth || uint32(y) >= clientHeight {
		return 0, 0, false
	}
	windowWidth, windowHeight := float64(clientWidth), float64(clientHeight)
	videoAspect := float64(videoWidth) / float64(videoHeight)
	viewportWidth, viewportHeight := windowWidth, windowHeight
	if !fill {
		if videoAspect > windowWidth/windowHeight {
			viewportHeight = viewportWidth / videoAspect
		} else {
			viewportWidth = viewportHeight * videoAspect
		}
	}
	offsetX, offsetY := (windowWidth-viewportWidth)/2, (windowHeight-viewportHeight)/2
	if float64(x) < offsetX || float64(y) < offsetY || float64(x) >= offsetX+viewportWidth || float64(y) >= offsetY+viewportHeight {
		return 0, 0, false
	}
	return int((float64(x) - offsetX) * float64(videoWidth) / viewportWidth), int((float64(y) - offsetY) * float64(videoHeight) / viewportHeight), true
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
	androidReady := false
	var inputSession *android.InputSession
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
			androidReady = true
			inputSession, _ = device.StartInput()
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
	mouseStartX, mouseStartY := 0, 0
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
				androidReady = true
				if inputSession != nil {
					_ = inputSession.Close()
				}
				inputSession, _ = device.StartInput()
				log.Printf("Android device ready: %s", device.Serial)
			} else {
				androidReady = false
				if inputSession != nil {
					_ = inputSession.Close()
					inputSession = nil
				}
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
				if _, _, ok := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode); ok {
					mouseStartX, mouseStartY = int(x), int(y)
				}
			}
		}
		if !mouseDown && mouseWasDown && androidReady {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				x1, y1, startOK := mapClientToVideo(int32(mouseStartX), int32(mouseStartY), clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode)
				x2, y2, endOK := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode)
				if startOK && endOK {
					if abs(x2-x1)+abs(y2-y1) < 8 {
						if inputSession != nil {
							if err := inputSession.Tap(x2, y2); err != nil {
								log.Printf("Android tap failed: %v", err)
							}
						}
					} else {
						if inputSession != nil {
							if err := inputSession.Swipe(x1, y1, x2, y2, 120); err != nil {
								log.Printf("Android swipe failed: %v", err)
							}
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
	if inputSession != nil {
		_ = inputSession.Close()
	}
	<-stopped
	if current != nil {
		buffers <- current.buffer
	}
}
