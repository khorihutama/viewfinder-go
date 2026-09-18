//go:build windows

package renderer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	d3d11DLL          = windows.NewLazySystemDLL("d3d11.dll")
	dxgiDLL           = windows.NewLazySystemDLL("dxgi.dll")
	d3d11CreateDevice = d3d11DLL.NewProc("D3D11CreateDevice")
)

const (
	d3d11SDKVersion = 7

	d3dDriverTypeHardware = 1

	d3dFeatureLevel11_0 = 0xB000
	d3dFeatureLevel11_1 = 0xB100
)

type Device struct {
	device       uintptr
	context      uintptr
	featureLevel uint32
}

func Initialize() (*Device, error) {
	var device uintptr
	var context uintptr
	var featureLevel uint32

	result, _, _ := d3d11CreateDevice.Call(
		0,
		d3dDriverTypeHardware,
		0,
		0,
		0,
		0,
		d3d11SDKVersion,
		uintptr(unsafe.Pointer(&device)),
		uintptr(unsafe.Pointer(&featureLevel)),
		uintptr(unsafe.Pointer(&context)),
	)

	if result != 0 {
		return nil, fmt.Errorf(
			"D3D11CreateDevice failed: 0x%08X",
			uint32(result),
		)
	}

	if device == 0 {
		return nil, fmt.Errorf(
			"D3D11CreateDevice returned null device",
		)
	}

	if context == 0 {
		return nil, fmt.Errorf(
			"D3D11CreateDevice returned null context",
		)
	}

	return &Device{
		device:       device,
		context:      context,
		featureLevel: featureLevel,
	}, nil
}

func (d *Device) DevicePointer() uintptr {
	if d == nil {
		return 0
	}

	return d.device
}

func (d *Device) ContextPointer() uintptr {
	if d == nil {
		return 0
	}

	return d.context
}

func (d *Device) FeatureLevel() uint32 {
	if d == nil {
		return 0
	}

	return d.featureLevel
}
