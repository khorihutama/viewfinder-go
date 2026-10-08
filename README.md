# Viewfinder Go

A Windows capture-card viewer written in Go, with native D3D11 rendering and Android/Samsung DeX input forwarding over ADB.

Viewfinder captures video from a Windows capture device, renders NV12 frames using Direct3D 11, and optionally forwards mouse and keyboard input to an Android device or Samsung DeX session.

## Features

- Capture video from Windows capture devices using Media Foundation.
- Render NV12 video frames using Direct3D 11.
- Enumerate capture devices and supported formats.
- Runtime capture-device switching.
- Fullscreen and fit/fill display modes.
- Forward mouse input to Android through ADB.
- Forward keyboard input including:
  - Letters and numbers
  - Navigation keys
  - Modifier keys
  - Function keys
  - Enter, Backspace, Escape, and Space
- Android device selection through `VIEWFINDER_ANDROID_DEVICE`.
- Automatic Samsung DeX display detection.
- Display FPS, capture resolution, and Android connection state.

## Architecture

The project uses Go for the application layer and native C++ for Windows-specific capture and rendering functionality.

```text
                         ┌─────────────────────┐
                         │      Go App         │
                         │                     │
                         │ Window / Input      │
                         │ Device Selection    │
                         │ Android / ADB       │
                         └──────────┬──────────┘
                                    │
                                    │ CGO / DLL
                                    ▼
                         ┌─────────────────────┐
                         │   Native C++ DLL    │
                         │                     │
                         │ Media Foundation    │
                         │ D3D11 Rendering     │
                         │ Capture Pipeline    │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │    Capture Device   │
                         └─────────────────────┘

Android Input
     │
     ▼
 Go Input Handling
     │
     ▼
    ADB
     │
     ▼
 Android / Samsung DeX
```

### Technology

- **Go 1.26+**
- **C++17**
- **Windows Media Foundation**
- **Direct3D 11**
- **ADB / Android SDK Platform-Tools**
- **Windows 10/11**

## Requirements

- Windows 10 or Windows 11
- Go 1.26+
- Visual Studio Build Tools with:
  - C++ build tools
  - Windows SDK
- Windows Media Foundation
- Android SDK Platform-Tools (`adb`) for Android input forwarding

## Build

The native component must be built using a Visual Studio Developer Command Prompt.

```powershell
call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\Common7\Tools\VsDevCmd.bat" -arch=x64 -host_arch=x64
```

Build the native DLL:

```powershell
cl /nologo /std:c++17 /EHsc /LD /DVIEWFINDER_NATIVE_EXPORTS `
  internal\native\capture.cpp internal\native\renderer.cpp `
  /I internal\native /Fe:build\viewfinder_native.dll `
  user32.lib d3d11.lib d3dcompiler.lib dxgi.lib `
  mfplat.lib mf.lib mfuuid.lib mfreadwrite.lib ole32.lib
```

Build the Go application:

```powershell
go build -o build\viewfinder-go.exe .
```

To package the native DLL, shaders, and executable into a standalone release directory:

```powershell
.\scripts\package-release.ps1
```

The following files must be available beside the executable:

```text
build/
├── viewfinder-go.exe
├── viewfinder_native.dll
├── fullscreen_vs.cso
└── nv12_ps.cso
```

## Run

```powershell
.\build\viewfinder-go.exe
```

On startup, Viewfinder enumerates available capture devices, opens the selected stream, and starts the D3D11 renderer.

## Android / Samsung DeX

Connect the Android device using either USB or ADB over Wi-Fi.

Verify the connection:

```powershell
adb devices
```

Viewfinder detects the external Samsung DeX display using:

```text
dumpsys display
```

Display `0` is treated as the built-in phone display.

### Select an Android device

If multiple devices are connected, set:

```powershell
$env:VIEWFINDER_ANDROID_DEVICE = "192.168.10.233:40445"
```

### Specify a display ID

If automatic display detection is not suitable for a particular device:

```powershell
$env:VIEWFINDER_ANDROID_DISPLAY_ID = "8"
```

## Controls

| Input | Action |
|---|---|
| Right-click | Open capture-device menu |
| `F11` | Toggle fullscreen |
| `F` | Toggle fit/fill mode |
| Left-click | Android tap |
| Left-button drag | Android touch/swipe |
| Mouse wheel | Android scroll |
| Keyboard | Forward supported keys to Android |

The window title displays the current capture resolution, FPS, and Android connection status.

## Current Limitations

- Android input currently uses a persistent ADB shell rather than the full `scrcpy` input protocol.
- Multi-touch is not implemented.
- Full Android text-input composition is not implemented.
- Capture format selection currently uses the first available format for the selected device.
- Runtime device selection is supported, but capture-format selection from the context menu is intentionally disabled.

## Project Status

Viewfinder is an active side project and is currently usable for capture-card viewing and basic Android/DeX input forwarding.

The project is still being developed, with input handling and capture-format selection being areas for future improvements.

## Why I Built This

Viewfinder started as an exploration of building a low-level Windows video viewer in Go while integrating native Windows APIs.

The project combines several areas that normally live at different abstraction levels:

- Go application development
- CGO/native C++ integration
- Windows Media Foundation
- Direct3D 11
- GPU-based video rendering
- Windows input handling
- Android ADB communication
- Samsung DeX display handling

The main goal was to understand how these components can work together in a single application while keeping the higher-level application logic in Go.
