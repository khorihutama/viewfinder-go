//go:build windows

package native

import (
	"fmt"
	"unsafe"
)

type Capture struct {
	ptr uintptr
}

var (
	procCaptureCreate = dll.NewProc(
		"ViewfinderCaptureCreate",
	)

	procCaptureInitialize = dll.NewProc(
		"ViewfinderCaptureInitialize",
	)

	procCaptureDestroy = dll.NewProc(
		"ViewfinderCaptureDestroy",
	)
)

func CreateCapture() (*Capture, error) {
	var capture uintptr

	result, _, _ := procCaptureCreate.Call(
		uintptr(unsafe.Pointer(&capture)),
	)

	hr := int32(uint32(result))

	if hr < 0 {
		return nil, fmt.Errorf(
			"ViewfinderCaptureCreate failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	if capture == 0 {
		return nil, fmt.Errorf(
			"native capture is nil",
		)
	}

	return &Capture{
		ptr: capture,
	}, nil
}

func (c *Capture) Initialize() error {
	if c == nil || c.ptr == 0 {
		return fmt.Errorf(
			"capture is nil",
		)
	}

	result, _, _ := procCaptureInitialize.Call(
		c.ptr,
	)

	hr := int32(uint32(result))

	if hr < 0 {
		return fmt.Errorf(
			"ViewfinderCaptureInitialize failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	return nil
}

func (c *Capture) Close() {
	if c == nil || c.ptr == 0 {
		return
	}

	procCaptureDestroy.Call(
		c.ptr,
	)

	c.ptr = 0
}
