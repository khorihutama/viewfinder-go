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

type CaptureFormat struct {
	Index uint32

	Width  uint32
	Height uint32

	FPSNumerator   uint32
	FPSDenominator uint32

	Subtype string
}

const maxDeviceName = 256
const maxSubtypeName = 64

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

	procCaptureGetFormatCount = dll.NewProc(
		"ViewfinderCaptureGetFormatCount",
	)

	procCaptureGetFormat = dll.NewProc(
		"ViewfinderCaptureGetFormat",
	)

	procCaptureDestroy = dll.NewProc(
		"ViewfinderCaptureDestroy",
	)

	procCaptureOpen = dll.NewProc(
		"ViewfinderCaptureOpen",
	)

	procCaptureIsOpen = dll.NewProc(
		"ViewfinderCaptureIsOpen",
	)

	procCaptureClose = dll.NewProc(
		"ViewfinderCaptureClose",
	)
)

type nativeCaptureDevice struct {
	Index uint32
	Name  [maxDeviceName]uint16
}

type nativeCaptureFormat struct {
	Index uint32

	Width  uint32
	Height uint32

	FPSNumerator   uint32
	FPSDenominator uint32

	Subtype [maxSubtypeName]uint16
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

func (c *Capture) Device(
	index uint32,
) (*CaptureDevice, error) {
	if c == nil || c.ptr == 0 {
		return nil, fmt.Errorf(
			"capture is nil",
		)
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

func (c *Capture) FormatCount(
	deviceIndex uint32,
) (uint32, error) {
	if c == nil || c.ptr == 0 {
		return 0, fmt.Errorf(
			"capture is nil",
		)
	}

	var count uint32

	result, _, _ := procCaptureGetFormatCount.Call(
		c.ptr,
		uintptr(deviceIndex),
		uintptr(unsafe.Pointer(&count)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetFormatCount",
	); err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Capture) Format(
	deviceIndex uint32,
	formatIndex uint32,
) (*CaptureFormat, error) {
	if c == nil || c.ptr == 0 {
		return nil, fmt.Errorf(
			"capture is nil",
		)
	}

	var nativeFormat nativeCaptureFormat

	result, _, _ := procCaptureGetFormat.Call(
		c.ptr,
		uintptr(deviceIndex),
		uintptr(formatIndex),
		uintptr(unsafe.Pointer(&nativeFormat)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetFormat",
	); err != nil {
		return nil, err
	}

	subtype := syscall.UTF16ToString(
		nativeFormat.Subtype[:],
	)

	return &CaptureFormat{
		Index: nativeFormat.Index,

		Width:  nativeFormat.Width,
		Height: nativeFormat.Height,

		FPSNumerator:   nativeFormat.FPSNumerator,
		FPSDenominator: nativeFormat.FPSDenominator,

		Subtype: subtype,
	}, nil
}

func (c *Capture) Formats(
	deviceIndex uint32,
) ([]CaptureFormat, error) {
	count, err := c.FormatCount(
		deviceIndex,
	)

	if err != nil {
		return nil, err
	}

	formats := make(
		[]CaptureFormat,
		0,
		count,
	)

	for i := uint32(0); i < count; i++ {
		format, err := c.Format(
			deviceIndex,
			i,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"get format %d: %w",
				i,
				err,
			)
		}

		formats = append(
			formats,
			*format,
		)
	}

	return formats, nil
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

func (c *Capture) Open(
	deviceIndex uint32,
	formatIndex uint32,
) error {
	if c == nil || c.ptr == 0 {
		return fmt.Errorf("capture is nil")
	}

	result, _, _ := procCaptureOpen.Call(
		c.ptr,
		uintptr(deviceIndex),
		uintptr(formatIndex),
	)

	return checkHRESULT(
		result,
		"ViewfinderCaptureOpen",
	)
}

func (c *Capture) IsOpen() bool {
	if c == nil || c.ptr == 0 {
		return false
	}

	result, _, _ := procCaptureIsOpen.Call(
		c.ptr,
	)

	return int32(result) == 1
}

func (c *Capture) CloseStream() {
	if c == nil || c.ptr == 0 {
		return
	}

	procCaptureClose.Call(
		c.ptr,
	)
}
