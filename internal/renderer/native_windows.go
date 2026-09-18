//go:build windows

package renderer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var dll = windows.NewLazyDLL(
	"build\\viewfinder_native.dll",
)

var (
	procRendererInitialize = dll.NewProc(
		"ViewfinderRendererInitialize",
	)

	procRendererClear = dll.NewProc(
		"ViewfinderRendererClear",
	)

	procRendererPresent = dll.NewProc(
		"ViewfinderRendererPresent",
	)

	procRendererDestroy = dll.NewProc(
		"ViewfinderRendererDestroy",
	)
)

func InitializeWindow(
	hwnd uintptr,
	width uint32,
	height uint32,
) error {
	if hwnd == 0 {
		return fmt.Errorf("invalid HWND")
	}

	result, _, _ := procRendererInitialize.Call(
		hwnd,
		uintptr(width),
		uintptr(height),
	)

	if int32(result) < 0 {
		return fmt.Errorf(
			"ViewfinderRendererInitialize failed: 0x%08X",
			uint32(result),
		)
	}

	return nil
}

func Clear(
	red float32,
	green float32,
	blue float32,
	alpha float32,
) error {
	result, _, _ := procRendererClear.Call(
		uintptr(*(*uint32)(unsafe.Pointer(&red))),
		uintptr(*(*uint32)(unsafe.Pointer(&green))),
		uintptr(*(*uint32)(unsafe.Pointer(&blue))),
		uintptr(*(*uint32)(unsafe.Pointer(&alpha))),
	)

	if int32(result) < 0 {
		return fmt.Errorf(
			"ViewfinderRendererClear failed: 0x%08X",
			uint32(result),
		)
	}

	return nil
}

func Present() error {
	result, _, _ := procRendererPresent.Call()

	if int32(result) < 0 {
		return fmt.Errorf(
			"ViewfinderRendererPresent failed: 0x%08X",
			uint32(result),
		)
	}

	return nil
}

func DestroyWindowRenderer() {
	procRendererDestroy.Call()
}
