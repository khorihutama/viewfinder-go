//go:build windows

package renderer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	driverTypeHardware = 1

	d3d11SDKVersion = 7

	createDeviceBGRA = 0x20

	featureLevel11_0 = 0xB000
	featureLevel11_1 = 0xB100
)

var (
	d3d11DLL = windows.NewLazySystemDLL("d3d11.dll")

	d3d11CreateDevice = d3d11DLL.NewProc("D3D11CreateDevice")
)

// Device represents the Direct3D 11 device and immediate context.
type Device struct {
	device       uintptr
	context      uintptr
	featureLevel uint32
}

// Initialize creates a hardware-accelerated Direct3D 11 device.
func Initialize() (*Device, error) {
	featureLevels := []uint32{
		featureLevel11_1,
		featureLevel11_0,
	}

	var (
		device       uintptr
		context      uintptr
		featureLevel uint32
	)

	hr, _, _ := d3d11CreateDevice.Call(
		0, // pAdapter - default adapter
		driverTypeHardware,
		0, // Software
		createDeviceBGRA,
		uintptr(unsafe.Pointer(&featureLevels[0])),
		uintptr(len(featureLevels)),
		d3d11SDKVersion,
		uintptr(unsafe.Pointer(&device)),
		uintptr(unsafe.Pointer(&featureLevel)),
		uintptr(unsafe.Pointer(&context)),
	)

	if hr != 0 {
		return nil, fmt.Errorf(
			"D3D11CreateDevice failed: HRESULT 0x%08X",
			uint32(hr),
		)
	}

	if device == 0 {
		return nil, fmt.Errorf("D3D11CreateDevice returned a nil device")
	}

	if context == 0 {
		return nil, fmt.Errorf("D3D11CreateDevice returned a nil context")
	}

	return &Device{
		device:       device,
		context:      context,
		featureLevel: featureLevel,
	}, nil
}

// DevicePointer returns the native ID3D11Device pointer.
//
// This is primarily useful internally when interacting with
// other Direct3D/DXGI COM interfaces.
func (d *Device) DevicePointer() uintptr {
	if d == nil {
		return 0
	}

	return d.device
}

// ContextPointer returns the native ID3D11DeviceContext pointer.
func (d *Device) ContextPointer() uintptr {
	if d == nil {
		return 0
	}

	return d.context
}

// FeatureLevel returns the Direct3D feature level selected by the driver.
func (d *Device) FeatureLevel() uint32 {
	if d == nil {
		return 0
	}

	return d.featureLevel
}
