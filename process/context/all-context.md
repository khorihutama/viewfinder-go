# Viewfinder Go Context

## Project Description

Viewfinder Go is a Windows-first capture-card viewer that lets a broader Android/Samsung DeX audience see HDMI/UVC video and control the displayed Android device. The project prioritizes low-latency input, capture reliability, keyboard support, and polished UI. ADB is the first Android transport. Linux and macOS are future targets.

## Quick Start

Read this file first. Use `process/context/tests/all-tests.md` for validation limits and manual test coverage. Source ownership is split between `main.go`, `internal/native`, `internal/win32`, `internal/renderer`, and `internal/android`.

## Current Root Entry Points

| Area | Entry point |
| --- | --- |
| Project context | `process/context/all-context.md` |
| Test and validation context | `process/context/tests/all-tests.md` |
| Planning context | `process/context/planning/` |
| Runtime entry point | `main.go` |

## Current Context Groups

| Group | Path | Scope |
| --- | --- | --- |
| Tests | `process/context/tests/all-tests.md` | Build, manual hardware checks, and known gaps |

## Repository Structure

```text
main.go                         Go application loop and orchestration
go.mod                          Go module and dependency declaration
internal/android/               ADB discovery, DeX display, persistent input
internal/native/                Media Foundation capture and D3D11 C++ DLL
internal/renderer/              Go renderer syscall wrapper
internal/win32/                 Win32 window, menu, input, fullscreen
build/                          Generated executable, DLL, and shader binaries
README.md                       Build and runtime documentation
```

## Technology Stack

- Go 1.26.0 with standard library and `golang.org/x/sys`.
- Windows Win32 APIs through `golang.org/x/sys/windows`.
- C++17 native DLL compiled with Visual Studio Build Tools.
- Media Foundation and Source Reader for UVC capture devices.
- D3D11, DXGI, HLSL, and GPU NV12-to-RGB conversion.
- ADB over USB or Wi-Fi for Android input.
- Samsung DeX external display targeting; display 0 is treated as built-in phone display.

## Architecture and Data Flow

Capture worker reads Media Foundation frames into two reusable Go buffers. A one-frame bounded queue drops stale frames. The main OS-thread loop pumps Win32 messages, updates the renderer, and presents the newest NV12 frame. Input uses a persistent ADB shell session and maps Windows client coordinates into capture coordinates.

## Key Patterns and Conventions

- Keep Windows-specific Go files behind `//go:build windows`.
- Keep native ownership and HRESULT handling inside the native wrapper boundary.
- Prefer bounded queues and buffer reuse for low latency.
- Use explicit cleanup for capture, renderer, worker, and ADB sessions.
- Preserve minimal dependencies; do not add a framework for UI or input.
- Treat external DeX display as the default Android input target, with environment fallback for display ID.
- Manual hardware validation is currently required; unit-test pure coordinate/display parsing helpers as they are extracted.

## Environment and Configuration

| Variable | Purpose |
| --- | --- |
| `VIEWFINDER_ANDROID_DEVICE` | Select an ADB serial at startup |
| `VIEWFINDER_ANDROID_DISPLAY_ID` | Fallback external display ID |
| `VIEWFINDER_CAPTURE_DEVICE` | Select capture device by index or exact name |

## Priorities and Known Gaps

1. Low-latency input, eventually replacing shell commands with a scrcpy-compatible binary protocol.
2. Capture reconnect and `0xC00D3704` recovery validation.
3. Keyboard completeness, text composition, and key repeat.
4. UI polish and device/status controls.
5. Linux/macOS portability after Windows behavior is stable.

Current limitations include no multi-touch, no full text-input composition, no automated hardware test suite, and first-format selection for runtime capture changes.

## Scan Metadata

- Generated: 2026-09-23
- HEAD: 697cf81
- Mode: fresh
- Package manager: Go modules
