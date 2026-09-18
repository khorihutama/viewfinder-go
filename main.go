package main

import (
	"log"
	"runtime"
	"time"

	"github.com/khorihutama/viewfinder-go/internal/native"
	"github.com/khorihutama/viewfinder-go/internal/win32"
)

const (
	windowWidth  = 1280
	windowHeight = 720

	targetFPS = 60
)

func main() {
	// D3D11 and Win32 operations are kept on one OS thread.
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

	frameDuration := time.Second / targetFPS

	nextFrame := time.Now()

	var frame uint64

	for {
		// Process all pending Windows messages.
		running, err := window.Pump()

		if err != nil {
			log.Fatalf(
				"message pump: %v",
				err,
			)
		}

		if !running {
			log.Println("Window closed")
			return
		}

		now := time.Now()

		if now.Before(nextFrame) {
			// Give the Go scheduler some time.
			runtime.Gosched()
			continue
		}

		frame++

		if err := renderer.Render(); err != nil {
			log.Fatalf(
				"render frame %d: %v",
				frame,
				err,
			)
		}

		nextFrame = nextFrame.Add(
			frameDuration,
		)

		// Recover if rendering fell significantly behind.
		if nextFrame.Before(now) {
			nextFrame = now.Add(frameDuration)
		}
	}
}
