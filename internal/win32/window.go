//go:build windows

package win32

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_POPUP            = 0x80000000
	CW_USEDEFAULT       = 0x80000000

	SW_SHOW = 5

	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_QUIT          = 0x0012
	WM_KEYDOWN       = 0x0100
	WM_MOUSEWHEEL    = 0x020A
	VK_LBUTTON       = 0x01
	VK_BACK          = 0x08
	VK_RETURN        = 0x0D
	VK_ESCAPE        = 0x1B
	VK_SPACE         = 0x20
	VK_PRIOR         = 0x21
	VK_NEXT          = 0x22
	VK_END           = 0x23
	VK_HOME          = 0x24
	VK_LEFT          = 0x25
	VK_UP            = 0x26
	VK_RIGHT         = 0x27
	VK_DOWN          = 0x28
	VK_TAB           = 0x09
	VK_SHIFT         = 0x10
	VK_CONTROL       = 0x11
	VK_MENU          = 0x12
	VK_F1            = 0x70
	VK_F11           = 0x7A
	GWL_STYLE        = -16
	SWP_FRAMECHANGED = 0x0020
	SWP_NOZORDER     = 0x0004
	SWP_SHOWWINDOW   = 0x0040
	SM_CXSCREEN      = 0
	SM_CYSCREEN      = 1

	PM_REMOVE = 0x0001
)

type point struct {
	X int32
	Y int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")

	procShowWindow   = user32.NewProc("ShowWindow")
	procUpdateWindow = user32.NewProc("UpdateWindow")

	procGetMessageW      = user32.NewProc("GetMessageW")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")

	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	procGetAsyncKeyState  = user32.NewProc("GetAsyncKeyState")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
	procScreenToClient    = user32.NewProc("ScreenToClient")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	windowProc = windows.NewCallback(wndProc)
)

type Window struct {
	HWND       uintptr
	fullscreen bool
	style      uintptr
	windowRect rect
}

var wheelDelta int32

func (w *Window) ConsumeWheel() int32 {
	delta := wheelDelta
	wheelDelta = 0
	return delta
}

func (w *Window) KeyDown(key int) bool {
	result, _, _ := procGetAsyncKeyState.Call(uintptr(key))
	return int16(result) < 0
}

func (w *Window) CursorClient() (int32, int32, bool) {
	var p point
	if result, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); result == 0 {
		return 0, 0, false
	}
	if result, _, _ := procScreenToClient.Call(w.HWND, uintptr(unsafe.Pointer(&p))); result == 0 {
		return 0, 0, false
	}
	return p.X, p.Y, true
}

func (w *Window) ToggleFullscreen() error {
	if !w.fullscreen {
		var old rect
		if result, _, err := procGetWindowRect.Call(w.HWND, uintptr(unsafe.Pointer(&old))); result == 0 {
			return fmt.Errorf("GetWindowRect failed: %w", err)
		}
		w.style, _, _ = procGetWindowLongPtrW.Call(w.HWND, ^uintptr(15))
		w.windowRect = old
		procSetWindowLongPtrW.Call(w.HWND, ^uintptr(15), WS_POPUP)
		width, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
		height, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
		procSetWindowPos.Call(w.HWND, 0, 0, 0, width, height, SWP_FRAMECHANGED|SWP_NOZORDER|SWP_SHOWWINDOW)
		w.fullscreen = true
		return nil
	}
	procSetWindowLongPtrW.Call(w.HWND, ^uintptr(15), w.style)
	width := uintptr(w.windowRect.Right - w.windowRect.Left)
	height := uintptr(w.windowRect.Bottom - w.windowRect.Top)
	procSetWindowPos.Call(w.HWND, 0, uintptr(w.windowRect.Left), uintptr(w.windowRect.Top), width, height, SWP_FRAMECHANGED|SWP_NOZORDER|SWP_SHOWWINDOW)
	w.fullscreen = false
	return nil
}

func (w *Window) SetTitle(title string) error {
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	result, _, callErr := procSetWindowTextW.Call(
		w.HWND,
		uintptr(unsafe.Pointer(titlePtr)),
	)
	if result == 0 {
		return fmt.Errorf("SetWindowTextW failed: %w", callErr)
	}
	return nil
}

func Create(title string, width, height int) (*Window, error) {
	className, err := windows.UTF16PtrFromString(
		"ViewfinderGoWindow",
	)
	if err != nil {
		return nil, err
	}

	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return nil, err
	}

	hInstance, _, err := procGetModuleHandleW.Call(0)

	if hInstance == 0 {
		return nil, fmt.Errorf(
			"GetModuleHandleW failed: %w",
			err,
		)
	}

	class := wndClassEx{
		CbSize: uint32(
			unsafe.Sizeof(wndClassEx{}),
		),

		Style: CS_HREDRAW | CS_VREDRAW,

		LpfnWndProc: windowProc,

		HInstance: windows.Handle(hInstance),

		// COLOR_WINDOW + 1
		HbrBackground: windows.Handle(6),

		LpszClassName: className,
	}

	atom, _, err := procRegisterClassExW.Call(
		uintptr(unsafe.Pointer(&class)),
	)

	if atom == 0 && err != windows.ERROR_CLASS_ALREADY_EXISTS {
		return nil, fmt.Errorf(
			"RegisterClassExW failed: %w",
			err,
		)
	}

	hwnd, _, err := procCreateWindowExW.Call(
		0,

		uintptr(unsafe.Pointer(className)),

		uintptr(unsafe.Pointer(titlePtr)),

		WS_OVERLAPPEDWINDOW,

		CW_USEDEFAULT,
		CW_USEDEFAULT,

		uintptr(width),
		uintptr(height),

		0,
		0,

		hInstance,

		0,
	)

	if hwnd == 0 {
		return nil, fmt.Errorf(
			"CreateWindowExW failed: %w",
			err,
		)
	}

	procShowWindow.Call(
		hwnd,
		SW_SHOW,
	)

	procUpdateWindow.Call(
		hwnd,
	)

	return &Window{
		HWND: hwnd,
	}, nil
}

// Pump processes all currently pending Windows messages.
//
// It does not block, which allows the renderer to continue running.
func (w *Window) Pump() (bool, error) {
	var message msg

	for {
		ret, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
			0,
			0,
			0,
			PM_REMOVE,
		)

		if ret == 0 {
			return true, nil
		}

		if message.Message == WM_QUIT {
			return false, nil
		}
		if message.Message == WM_KEYDOWN && message.WParam == VK_F11 {
			if err := w.ToggleFullscreen(); err != nil {
				return false, err
			}
			continue
		}

		procTranslateMessage.Call(
			uintptr(unsafe.Pointer(&message)),
		)

		procDispatchMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
		)
	}
}

func (w *Window) ClientSize() (uint32, uint32, error) {
	var client rect

	result, _, err := procGetClientRect.Call(
		w.HWND,
		uintptr(unsafe.Pointer(&client)),
	)
	if result == 0 {
		return 0, 0, fmt.Errorf("GetClientRect failed: %w", err)
	}

	return uint32(client.Right - client.Left),
		uint32(client.Bottom - client.Top),
		nil
}

// Run provides a traditional blocking Windows message loop.
func (w *Window) Run() error {
	var message msg

	for {
		ret, _, err := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
			0,
			0,
			0,
		)

		if int32(ret) == -1 {
			return fmt.Errorf(
				"GetMessageW failed: %w",
				err,
			)
		}

		if ret == 0 {
			break
		}

		procTranslateMessage.Call(
			uintptr(unsafe.Pointer(&message)),
		)

		procDispatchMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
		)
	}

	return nil
}

func wndProc(
	hwnd uintptr,
	message uint32,
	wParam uintptr,
	lParam uintptr,
) uintptr {
	switch message {
	case WM_MOUSEWHEEL:
		wheelDelta += int32(int16((wParam >> 16) & 0xffff))
		return 0

	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(
		hwnd,
		uintptr(message),
		wParam,
		lParam,
	)

	return ret
}
