# Viewfinder Go

Windows capture-card viewer with D3D11 NV12 rendering and Android/Samsung DeX input forwarding over ADB.

## Requirements

- Windows 10/11
- Go 1.26+
- Visual Studio Build Tools with C++ and Windows SDK
- Media Foundation
- Android SDK Platform-Tools (`adb`) for Android input

## Build

Build the native DLL from a Visual Studio Developer Command Prompt:

```powershell
call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\Common7\Tools\VsDevCmd.bat" -arch=x64 -host_arch=x64
cl /nologo /std:c++17 /EHsc /LD /DVIEWFINDER_NATIVE_EXPORTS `
  internal\native\capture.cpp internal\native\renderer.cpp `
  /I internal\native /Fe:build\viewfinder_native.dll `
  user32.lib d3d11.lib d3dcompiler.lib dxgi.lib `
  mfplat.lib mf.lib mfuuid.lib mfreadwrite.lib ole32.lib
```

Build the Go executable:

```powershell
go build -o build\viewfinder-go.exe .
```

Create a standalone release folder after building native and Go artifacts:

```powershell
.\scripts\package-release.ps1
```

Keep `viewfinder_native.dll`, `fullscreen_vs.cso`, and `nv12_ps.cso` beside the executable in `build\`.

## Run

```powershell
.\build\viewfinder-go.exe
```

The application enumerates capture devices and formats, opens the selected stream, and starts the D3D11 renderer.

## Android / Samsung DeX

Connect Android over ADB Wi-Fi or USB, then verify:

```powershell
adb devices
```

The application discovers the external DeX display ID from `dumpsys display`. Display `0` is treated as the built-in phone display.

Optional device selection:

```powershell
$env:VIEWFINDER_ANDROID_DEVICE = "192.168.10.233:40445"
```

Optional display-ID fallback:

```powershell
$env:VIEWFINDER_ANDROID_DISPLAY_ID = "8"
```

## Runtime controls

- Right-click: open capture-device menu.
- `F11`: toggle fullscreen.
- `F`: toggle fit/fill mode.
- Left-click: Android tap.
- Left-button drag: Android touch/swipe.
- Mouse wheel: Android scroll.
- Keyboard: common letters, numbers, navigation, modifiers, function keys, Enter, Backspace, Escape, and Space are forwarded to Android.

The window title shows capture resolution, FPS, and Android connection state.

## Current limitations

- Input uses a persistent ADB shell, not the full scrcpy binary input protocol.
- Multi-touch and full text-input composition are not implemented.
- Capture format selection currently uses the first format for a selected device.
- Runtime capture-device selection is available from the context menu; format submenu is intentionally disabled.
