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

	procClear = dll.NewProc(
		"ViewfinderClear",
	)

	procPresent = dll.NewProc(
		"ViewfinderPresent",
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

	r1, _, _ := procCreateRenderer.Call(
		hwnd,
		uintptr(width),
		uintptr(height),
		uintptr(unsafe.Pointer(&renderer)),
	)

	hr := int32(uint32(r1))

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

func (r *Renderer) Clear(
	red float32,
	green float32,
	blue float32,
	alpha float32,
) error {
	if r == nil || r.ptr == 0 {
		return fmt.Errorf(
			"renderer is nil",
		)
	}

	r1, _, _ := procClear.Call(
		r.ptr,
		uintptr(*(*uint32)(unsafe.Pointer(&red))),
		uintptr(*(*uint32)(unsafe.Pointer(&green))),
		uintptr(*(*uint32)(unsafe.Pointer(&blue))),
		uintptr(*(*uint32)(unsafe.Pointer(&alpha))),
	)

	hr := int32(uint32(r1))

	if hr < 0 {
		return fmt.Errorf(
			"ViewfinderClear failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	return nil
}

func (r *Renderer) Present() error {
	if r == nil || r.ptr == 0 {
		return fmt.Errorf(
			"renderer is nil",
		)
	}

	r1, _, _ := procPresent.Call(
		r.ptr,
	)

	hr := int32(uint32(r1))

	if hr < 0 {
		return fmt.Errorf(
			"ViewfinderPresent failed: HRESULT 0x%08X",
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
