//go:build windows

package native

import (
	"fmt"
	"syscall"
	"unsafe"
)

type Capture struct {
	ptr uintptr
}

type CaptureDevice struct {
	Index uint32
	Name  string
}

const maxDeviceName = 256

var (
	procCaptureCreate = dll.NewProc(
		"ViewfinderCaptureCreate",
	)

	procCaptureInitialize = dll.NewProc(
		"ViewfinderCaptureInitialize",
	)

	procCaptureGetDeviceCount = dll.NewProc(
		"ViewfinderCaptureGetDeviceCount",
	)

	procCaptureGetDevice = dll.NewProc(
		"ViewfinderCaptureGetDevice",
	)

	procCaptureDestroy = dll.NewProc(
		"ViewfinderCaptureDestroy",
	)
)

type nativeCaptureDevice struct {
	Index uint32

	Name [maxDeviceName]uint16
}

func CreateCapture() (*Capture, error) {
	var capture uintptr

	result, _, _ := procCaptureCreate.Call(
		uintptr(unsafe.Pointer(&capture)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureCreate",
	); err != nil {
		return nil, err
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

	return checkHRESULT(
		result,
		"ViewfinderCaptureInitialize",
	)
}

func (c *Capture) DeviceCount() (uint32, error) {
	if c == nil || c.ptr == 0 {
		return 0, fmt.Errorf(
			"capture is nil",
		)
	}

	var count uint32

	result, _, _ := procCaptureGetDeviceCount.Call(
		c.ptr,
		uintptr(unsafe.Pointer(&count)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetDeviceCount",
	); err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Capture) Device(index uint32) (*CaptureDevice, error) {
	if c == nil || c.ptr == 0 {
		return nil, fmt.Errorf("capture is nil")
	}

	var nativeDevice nativeCaptureDevice

	result, _, _ := procCaptureGetDevice.Call(
		c.ptr,
		uintptr(index),
		uintptr(unsafe.Pointer(&nativeDevice)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetDevice",
	); err != nil {
		return nil, err
	}

	name := syscall.UTF16ToString(
		nativeDevice.Name[:],
	)

	return &CaptureDevice{
		Index: nativeDevice.Index,
		Name:  name,
	}, nil
}

func (c *Capture) Devices() ([]CaptureDevice, error) {
	count, err := c.DeviceCount()

	if err != nil {
		return nil, err
	}

	devices := make(
		[]CaptureDevice,
		0,
		count,
	)

	for i := uint32(0); i < count; i++ {
		device, err := c.Device(i)

		if err != nil {
			return nil, fmt.Errorf(
				"get device %d: %w",
				i,
				err,
			)
		}

		devices = append(
			devices,
			*device,
		)
	}

	return devices, nil
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

func checkHRESULT(
	result uintptr,
	operation string,
) error {
	hr := int32(uint32(result))

	if hr < 0 {
		return fmt.Errorf(
			"%s failed: HRESULT 0x%08X",
			operation,
			uint32(hr),
		)
	}

	return nil
}
