//go:build windows

package native

import (
	"fmt"
	"syscall"
	"unsafe"
)

type Renderer struct {
	ptr uintptr
}

var (
	dll = syscall.NewLazyDLL(
		"viewfinder_native.dll",
	)

	procCreateRenderer = dll.NewProc(
		"ViewfinderCreateRenderer",
	)

	procRender = dll.NewProc(
		"ViewfinderRender",
	)

	procDestroyRenderer = dll.NewProc(
		"ViewfinderDestroyRenderer",
	)
)

func CreateRenderer(
	hwnd uintptr,
	width int,
	height int,
) (*Renderer, error) {
	if hwnd == 0 {
		return nil, fmt.Errorf(
			"invalid HWND",
		)
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf(
			"invalid dimensions: %dx%d",
			width,
			height,
		)
	}

	var renderer uintptr

	result, _, _ := procCreateRenderer.Call(
		hwnd,
		uintptr(width),
		uintptr(height),
		uintptr(unsafe.Pointer(&renderer)),
	)

	hr := int32(uint32(result))

	if hr < 0 {
		return nil, fmt.Errorf(
			"ViewfinderCreateRenderer failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	if renderer == 0 {
		return nil, fmt.Errorf(
			"native renderer is nil",
		)
	}

	return &Renderer{
		ptr: renderer,
	}, nil
}

func (r *Renderer) Render() error {
	if r == nil || r.ptr == 0 {
		return fmt.Errorf(
			"renderer is nil",
		)
	}

	result, _, _ := procRender.Call(
		r.ptr,
	)

	hr := int32(uint32(result))

	if hr < 0 {
		return fmt.Errorf(
			"ViewfinderRender failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	return nil
}

func (r *Renderer) Close() {
	if r == nil || r.ptr == 0 {
		return
	}

	procDestroyRenderer.Call(
		r.ptr,
	)

	r.ptr = 0
}
