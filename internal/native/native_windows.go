//go:build windows

package native

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	dll = syscall.NewLazyDLL("viewfinder_native.dll")

	procCreateSwapChain = dll.NewProc(
		"ViewfinderCreateSwapChain",
	)

	procReleaseObject = dll.NewProc(
		"ViewfinderReleaseObject",
	)
)

func CreateSwapChain(
	hwnd uintptr,
	width int,
	height int,
) (uintptr, error) {
	if hwnd == 0 {
		return 0, fmt.Errorf("invalid HWND")
	}

	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf(
			"invalid dimensions: %dx%d",
			width,
			height,
		)
	}

	var swapChain uintptr

	r1, _, _ := procCreateSwapChain.Call(
		hwnd,
		uintptr(width),
		uintptr(height),
		uintptr(unsafe.Pointer(&swapChain)),
	)

	hr := int32(uint32(r1))

	if hr < 0 {
		return 0, fmt.Errorf(
			"ViewfinderCreateSwapChain failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	if swapChain == 0 {
		return 0, fmt.Errorf(
			"native swap chain is nil",
		)
	}

	return swapChain, nil
}

func ReleaseObject(object uintptr) {
	if object == 0 {
		return
	}

	procReleaseObject.Call(object)
}
