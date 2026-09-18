//go:build windows

package native

import (
	"fmt"
	"syscall"
	"unsafe"
)

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

	procCaptureOpen = dll.NewProc(
		"ViewfinderCaptureOpen",
	)

	procCaptureIsOpen = dll.NewProc(
		"ViewfinderCaptureIsOpen",
	)

	procCaptureClose = dll.NewProc(
		"ViewfinderCaptureClose",
	)

	procCaptureReadFrame = dll.NewProc(
		"ViewfinderCaptureReadFrame",
	)
)

type Capture struct {
	ptr uintptr
}

type CaptureDevice struct {
	Index uint32
	Name  string
}

type CaptureFormat struct {
	Index          uint32
	Width          uint32
	Height         uint32
	FPSNumerator   uint32
	FPSDenominator uint32
	Subtype        string
}

type nativeCaptureDevice struct {
	Index uint32
	Name  [256]uint16
}

type nativeCaptureFormat struct {
	Index          uint32
	Width          uint32
	Height         uint32
	FPSNumerator   uint32
	FPSDenominator uint32
	Subtype        [64]uint16
}

func checkHRESULT(result uintptr, name string) error {
	hr := int32(result)

	if hr < 0 {
		return fmt.Errorf(
			"%s failed: 0x%08X",
			name,
			uint32(result),
		)
	}

	return nil
}

func CreateCapture() (*Capture, error) {
	var ptr uintptr

	result, _, _ := procCaptureCreate.Call(
		uintptr(unsafe.Pointer(&ptr)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureCreate",
	); err != nil {
		return nil, err
	}

	if ptr == 0 {
		return nil, fmt.Errorf(
			"ViewfinderCaptureCreate returned null",
		)
	}

	return &Capture{
		ptr: ptr,
	}, nil
}

func (c *Capture) Initialize() error {
	if c == nil || c.ptr == 0 {
		return fmt.Errorf("capture is nil")
	}

	result, _, _ := procCaptureInitialize.Call(
		c.ptr,
	)

	return checkHRESULT(
		result,
		"ViewfinderCaptureInitialize",
	)
}

func (c *Capture) Destroy() {
	if c == nil || c.ptr == 0 {
		return
	}

	procCaptureDestroy.Call(c.ptr)

	c.ptr = 0
}

func (c *Capture) DeviceCount() (uint32, error) {
	if c == nil || c.ptr == 0 {
		return 0, fmt.Errorf("capture is nil")
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
		return nil, fmt.Errorf("capture is nil")
	}

	var device nativeCaptureDevice

	result, _, _ := procCaptureGetDevice.Call(
		c.ptr,
		uintptr(index),
		uintptr(unsafe.Pointer(&device)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetDevice",
	); err != nil {
		return nil, err
	}

	return &CaptureDevice{
		Index: device.Index,
		Name:  syscall.UTF16ToString(device.Name[:]),
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
			return nil, err
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
		return 0, fmt.Errorf("capture is nil")
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
		return nil, fmt.Errorf("capture is nil")
	}

	var format nativeCaptureFormat

	result, _, _ := procCaptureGetFormat.Call(
		c.ptr,
		uintptr(deviceIndex),
		uintptr(formatIndex),
		uintptr(unsafe.Pointer(&format)),
	)

	if err := checkHRESULT(
		result,
		"ViewfinderCaptureGetFormat",
	); err != nil {
		return nil, err
	}

	return &CaptureFormat{
		Index:          format.Index,
		Width:          format.Width,
		Height:         format.Height,
		FPSNumerator:   format.FPSNumerator,
		FPSDenominator: format.FPSDenominator,
		Subtype: syscall.UTF16ToString(
			format.Subtype[:],
		),
	}, nil
}

func (c *Capture) Formats(
	deviceIndex uint32,
) ([]CaptureFormat, error) {
	count, err := c.FormatCount(deviceIndex)

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
			return nil, err
		}

		formats = append(
			formats,
			*format,
		)
	}

	return formats, nil
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

func (c *Capture) Close() {
	if c == nil || c.ptr == 0 {
		return
	}

	procCaptureClose.Call(c.ptr)
}

func (c *Capture) ReadFrame(
	buffer []byte,
) (
	dataSize uint32,
	width uint32,
	height uint32,
	stride uint32,
	err error,
) {
	if c == nil || c.ptr == 0 {
		return 0, 0, 0, 0,
			fmt.Errorf("capture is nil")
	}

	if len(buffer) == 0 {
		return 0, 0, 0, 0,
			fmt.Errorf("buffer is empty")
	}

	var outDataSize uint32
	var outWidth uint32
	var outHeight uint32
	var outStride uint32

	result, _, _ := procCaptureReadFrame.Call(
		c.ptr,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&outDataSize)),
		uintptr(unsafe.Pointer(&outWidth)),
		uintptr(unsafe.Pointer(&outHeight)),
		uintptr(unsafe.Pointer(&outStride)),
	)

	if int32(result) < 0 {
		return 0, 0, 0, 0,
			fmt.Errorf(
				"ViewfinderCaptureReadFrame failed: 0x%08X",
				uint32(result),
			)
	}

	return outDataSize,
		outWidth,
		outHeight,
		outStride,
		nil
}
