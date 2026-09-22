//go:build windows

package renderer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var dll = windows.NewLazyDLL(
	"viewfinder_native.dll",
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

	procRendererResize = dll.NewProc(
		"ViewfinderRendererResize",
	)

	procRendererDestroy = dll.NewProc(
		"ViewfinderRendererDestroy",
	)

	procRendererUploadNV12 = dll.NewProc(
		"ViewfinderRendererUploadNV12",
	)

	procRendererDraw = dll.NewProc(
		"ViewfinderRendererDraw",
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

func Resize(width, height uint32) error {
	if width == 0 || height == 0 {
		return nil
	}

	result, _, _ := procRendererResize.Call(uintptr(width), uintptr(height))
	if int32(result) < 0 {
		return fmt.Errorf("ViewfinderRendererResize failed: 0x%08X", uint32(result))
	}

	return nil
}

func UploadNV12(data []byte, width, height, stride uint32) error {
	if len(data) == 0 {
		return fmt.Errorf("NV12 data is empty")
	}

	result, _, _ := procRendererUploadNV12.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(width),
		uintptr(height),
		uintptr(stride),
	)

	if int32(result) < 0 {
		return fmt.Errorf("ViewfinderRendererUploadNV12 failed: 0x%08X", uint32(result))
	}

	return nil
}

func Draw() error {
	result, _, _ := procRendererDraw.Call()

	if int32(result) < 0 {
		return fmt.Errorf("ViewfinderRendererDraw failed: 0x%08X", uint32(result))
	}

	return nil
}

func DestroyWindowRenderer() {
	procRendererDestroy.Call()
}
