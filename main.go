package main

import (
	"fmt"
	"log"
	"os"
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

func captureDeviceName(devices []native.CaptureDevice, index uint32) string {
	for _, device := range devices {
		if device.Index == index {
			return device.Name
		}
	}
	return "unknown"
}

type captureFrame struct {
	buffer []byte
	data   uint32
	width  uint32
	height uint32
	stride uint32
}

type captureSwitch struct{ deviceIndex, formatIndex uint32 }

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

func readFrames(capture *native.Capture, deviceIndex, formatIndex uint32, buffers chan []byte, frames chan captureFrame, switches <-chan captureSwitch, done <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)
	defer capture.Close()
	lastReadError := time.Time{}
	readErrors := 0
	nextReopen := time.Time{}
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
		case request := <-switches:
			capture.Close()
			if err := capture.Open(request.deviceIndex, request.formatIndex); err != nil {
				log.Printf("capture switch failed: %v", err)
			} else {
				readErrors = 0
				log.Printf("capture switched: device=%d format=%d", request.deviceIndex, request.formatIndex)
			}
		case buffer := <-buffers:
			if buffer == nil {
				return
			}
			data, width, height, stride, err := capture.ReadFrame(buffer)
			if err != nil {
				readErrors++
				if lastReadError.IsZero() || time.Since(lastReadError) >= time.Second {
					log.Printf("ReadFrame failed: %v", err)
					lastReadError = time.Now()
				}
				if !returnBuffer(buffer) {
					return
				}
				if readErrors >= 40 && time.Now().After(nextReopen) {
					nextReopen = time.Now().Add(2 * time.Second)
					capture.Close()
					reopenDevice, reopenFormat := deviceIndex, formatIndex
					if devices, enumerateErr := capture.Devices(); enumerateErr == nil && len(devices) > 0 {
						reopenDevice = devices[0].Index
						if formats, formatErr := capture.Formats(reopenDevice); formatErr == nil && len(formats) > 0 {
							reopenFormat = formats[0].Index
						}
					}
					if reopenErr := capture.Open(reopenDevice, reopenFormat); reopenErr != nil {
						log.Printf("capture reopen failed: %v", reopenErr)
					} else {
						log.Println("capture stream reopened")
						readErrors = 0
					}
				}
				select {
				case <-done:
					return
				case <-time.After(25 * time.Millisecond):
				}
				continue
			}
			lastReadError = time.Time{}
			readErrors = 0

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
	selectedAndroidSerial := os.Getenv("VIEWFINDER_ANDROID_DEVICE")
	var androidDevice android.Device
	var inputSession *android.InputSession
	if devices, err := android.Devices(); err != nil {
		log.Printf("ADB unavailable: %v", err)
	} else {
		for _, device := range devices {
			log.Printf("Android device: %s (%s)", device.Serial, device.State)
		}
		if len(devices) == 0 {
			log.Println("No Android devices connected")
		} else if device, ok := android.SelectReady(devices, selectedAndroidSerial); ok {
			if displayID, resolveErr := android.ResolveDisplayID(device); resolveErr == nil {
				device.Display = displayID
				log.Printf("Android external display: %d", displayID)
			} else {
				log.Printf("Android external display detection failed: %v", resolveErr)
				device.Display = -1
			}
			androidReady = true
			androidDevice = device
			inputSession, _ = device.StartInput()
			if _, err := device.Shell("echo", "viewfinder-connected"); err != nil {
				log.Printf("ADB device command failed: %v", err)
			} else {
				log.Printf("ADB command ready: %s", device.Serial)
			}
		} else if selectedAndroidSerial != "" {
			log.Printf("Selected Android device unavailable: %s", selectedAndroidSerial)
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

	selectedCapture := os.Getenv("VIEWFINDER_CAPTURE_DEVICE")
	deviceIndex := devices[0].Index
	if selectedCapture != "" {
		matched := false
		for _, device := range devices {
			if selectedCapture == device.Name || selectedCapture == fmt.Sprint(device.Index) {
				deviceIndex = device.Index
				matched = true
				break
			}
		}
		if !matched {
			log.Fatalf("selected capture device unavailable: %s", selectedCapture)
		}
	}

	log.Printf(
		"Using device [%d]: %s",
		deviceIndex,
		captureDeviceName(devices, deviceIndex),
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
	switches := make(chan captureSwitch, 1)
	done := make(chan struct{})
	stopped := make(chan struct{})
	buffers <- bufferA
	buffers <- bufferB
	go readFrames(capture, deviceIndex, formatIndex, buffers, frames, switches, done, stopped)

	renderWidth := uint32(1280)
	renderHeight := uint32(720)
	var current *captureFrame
	fillMode := false
	fillKeyWasDown := false
	mouseWasDown := false
	mouseStartX, mouseStartY := 0, 0
	lastMotionX, lastMotionY := 0, 0
	keyWasDown := map[int]bool{}
	rightWasDown := false
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
			if device, ok := android.SelectReady(devices, selectedAndroidSerial); ok {
				if displayID, resolveErr := android.ResolveDisplayID(device); resolveErr == nil {
					device.Display = displayID
					log.Printf("Android external display: %d", displayID)
				} else {
					device.Display = -1
					log.Printf("Android external display detection failed: %v", resolveErr)
				}
				androidReady = true
				androidDevice = device
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
		if androidReady && (inputSession == nil || !inputSession.Alive()) {
			inputSession, _ = androidDevice.StartInput()
		}
		fillKeyDown := window.KeyDown('F')
		if fillKeyDown && !fillKeyWasDown {
			fillMode = !fillMode
			if err := renderer.SetFillMode(fillMode); err != nil {
				log.Fatalf("renderer fill mode failed: %v", err)
			}
		}
		fillKeyWasDown = fillKeyDown
		rightDown := window.KeyDown(0x02)
		if rightDown && !rightWasDown {
			if refreshed, refreshErr := capture.Devices(); refreshErr == nil && len(refreshed) > 0 {
				devices = refreshed
			}
			if len(devices) == 0 {
				log.Println("capture menu: no capture devices available")
				rightWasDown = rightDown
				continue
			}
			items := make([]string, len(devices))
			for i, device := range devices {
				marker := ""
				if device.Index == deviceIndex {
					marker = "[current] "
				}
				items[i] = fmt.Sprintf("%s%d: %s", marker, device.Index, device.Name)
			}
			if choice, menuErr := window.ShowCaptureMenu(items); menuErr == nil && choice > 0 && choice <= len(devices) {
				selected := devices[choice-1]
				formats, formatErr := capture.Formats(selected.Index)
				if formatErr == nil && len(formats) > 0 {
					selectedFormat = formats[0]
					deviceIndex, formatIndex = selected.Index, selectedFormat.Index
					switches <- captureSwitch{deviceIndex, formatIndex}
				}
			}
		}
		rightWasDown = rightDown
		for _, key := range []struct {
			vk      int
			android string
		}{
			{win32.VK_RETURN, "KEYCODE_ENTER"},
			{win32.VK_BACK, "KEYCODE_DEL"},
			{win32.VK_ESCAPE, "KEYCODE_BACK"},
			{win32.VK_SPACE, "KEYCODE_SPACE"},
		} {
			down := window.KeyDown(key.vk)
			if down && !keyWasDown[key.vk] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent(key.android); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[key.vk] = down
		}
		for _, key := range []struct {
			vk      int
			android string
		}{
			{win32.VK_LEFT, "KEYCODE_DPAD_LEFT"},
			{win32.VK_RIGHT, "KEYCODE_DPAD_RIGHT"},
			{win32.VK_UP, "KEYCODE_DPAD_UP"},
			{win32.VK_DOWN, "KEYCODE_DPAD_DOWN"},
			{win32.VK_HOME, "KEYCODE_MOVE_HOME"},
			{win32.VK_END, "KEYCODE_MOVE_END"},
			{win32.VK_PRIOR, "KEYCODE_PAGE_UP"},
			{win32.VK_NEXT, "KEYCODE_PAGE_DOWN"},
		} {
			down := window.KeyDown(key.vk)
			if down && !keyWasDown[key.vk] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent(key.android); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[key.vk] = down
		}
		for key := 'A'; key <= 'Z'; key++ {
			down := window.KeyDown(int(key))
			if down && !keyWasDown[int(key)] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent("KEYCODE_" + string(key)); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[int(key)] = down
		}
		for key := '0'; key <= '9'; key++ {
			down := window.KeyDown(int(key))
			if down && !keyWasDown[int(key)] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent("KEYCODE_" + string(key)); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[int(key)] = down
		}
		for _, key := range []struct {
			vk      int
			android string
		}{
			{win32.VK_TAB, "KEYCODE_TAB"},
			{win32.VK_SHIFT, "KEYCODE_SHIFT_LEFT"},
			{win32.VK_CONTROL, "KEYCODE_CTRL_LEFT"},
			{win32.VK_MENU, "KEYCODE_ALT_LEFT"},
		} {
			down := window.KeyDown(key.vk)
			if down && !keyWasDown[key.vk] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent(key.android); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[key.vk] = down
		}
		for key := 0; key < 12; key++ {
			vk := win32.VK_F1 + key
			down := window.KeyDown(vk)
			if down && !keyWasDown[vk] && androidReady && inputSession != nil {
				if err := inputSession.KeyEvent(fmt.Sprintf("KEYCODE_F%d", key+1)); err != nil {
					log.Printf("Android key failed: %v", err)
				}
			}
			keyWasDown[vk] = down
		}
		mouseDown := window.KeyDown(win32.VK_LBUTTON)
		if mouseDown && !mouseWasDown && androidReady {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				if _, _, ok := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode); ok {
					mouseStartX, mouseStartY = int(x), int(y)
					lastMotionX, lastMotionY = mouseStartX, mouseStartY
					if inputSession != nil {
						if deviceX, deviceY, mapped := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode); mapped {
							if err := inputSession.Motion("DOWN", deviceX, deviceY); err != nil {
								log.Printf("Android touch down failed: %v", err)
							}
						}
					}
				}
			}
		}
		if mouseDown && mouseWasDown && androidReady && inputSession != nil {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				if deviceX, deviceY, mapped := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode); mapped && (abs(deviceX-lastMotionX)+abs(deviceY-lastMotionY) >= 2) {
					if err := inputSession.Motion("MOVE", deviceX, deviceY); err != nil {
						log.Printf("Android touch move failed: %v", err)
					}
					lastMotionX, lastMotionY = deviceX, deviceY
				}
			}
		}
		if !mouseDown && mouseWasDown && androidReady {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				_, _, startOK := mapClientToVideo(int32(mouseStartX), int32(mouseStartY), clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode)
				x2, y2, endOK := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode)
				if startOK && endOK {
					if inputSession != nil {
						if err := inputSession.Motion("UP", x2, y2); err != nil {
							log.Printf("Android touch up failed: %v", err)
						}
					} else {
						log.Printf("Android gesture dropped: input session unavailable")
					}
				}
			}
		}
		mouseWasDown = mouseDown
		if delta := window.ConsumeWheel(); delta != 0 && androidReady && inputSession != nil {
			if x, y, ok := window.CursorClient(); ok {
				clientWidth, clientHeight, _ := window.ClientSize()
				if deviceX, deviceY, mapped := mapClientToVideo(x, y, clientWidth, clientHeight, selectedFormat.Width, selectedFormat.Height, fillMode); mapped {
					if err := inputSession.Scroll(deviceX, deviceY, int(delta)); err != nil {
						log.Printf("Android scroll failed: %v", err)
					}
				}
			}
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
				androidStatus := "Android disconnected"
				if androidReady {
					androidStatus = "Android connected"
				}
				if err := window.SetTitle(fmt.Sprintf("Viewfinder Go - %dx%d - %d FPS - %s", current.width, current.height, fps, androidStatus)); err != nil {
					log.Printf("window title update failed: %v", err)
				}
			} else if err := window.SetTitle("Viewfinder Go - Capture reconnecting"); err != nil {
				log.Printf("window title update failed: %v", err)
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
