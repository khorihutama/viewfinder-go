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

	procRendererUploadNV12 = dll.NewProc(
		"ViewfinderRendererUploadNV12",
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

func UploadNV12(
	data []byte,
	width uint32,
	height uint32,
	stride uint32,
) error {
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
		return fmt.Errorf(
			"ViewfinderRendererUploadNV12 failed: 0x%08X",
			uint32(result),
		)
	}

	return nil
}
