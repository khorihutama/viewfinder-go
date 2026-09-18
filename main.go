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

	log.Println("Media Foundation capture object created")

	if err := capture.Initialize(); err != nil {
		log.Fatalf(
			"initialize Media Foundation: %v",
			err,
		)
	}

	log.Println("Media Foundation initialized")

	frameDuration := time.Second / targetFPS
	nextFrame := time.Now()

	for {
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
			runtime.Gosched()
			continue
		}

		if err := renderer.Render(); err != nil {
			log.Fatalf(
				"render: %v",
				err,
			)
		}

		nextFrame = nextFrame.Add(
			frameDuration,
		)

		if nextFrame.Before(now) {
			nextFrame = now.Add(
				frameDuration,
			)
		}
	}
}
