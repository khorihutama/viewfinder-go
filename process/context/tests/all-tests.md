# Viewfinder Go Test Context

## Quick Decision Guide

No automated test runner or test files currently exist. Use `go build` for compile validation, `git diff --check` for whitespace validation, and manual hardware checks for capture, rendering, ADB, and input.

## Commands

```powershell
gofmt -w main.go internal\android\adb_windows.go internal\renderer\native_windows.go internal\win32\window.go
git diff --check
go build -o build\viewfinder-go.exe .
```

Native DLL build is documented in `README.md` and requires a Visual Studio Developer Command Prompt.

## Manual Hardware Checks

- Start with a connected UVC capture card and confirm device/format enumeration.
- Confirm 1920x1080 NV12 frames render and window resize preserves aspect ratio.
- Unplug/replug capture hardware and verify stream recovery without process restart.
- Connect Android/DeX through ADB Wi-Fi and verify external display ID discovery.
- Test tap, drag/swipe, scroll, keyboard, reconnect, and persistent-input latency.
- Verify right-click capture-device menu refresh and runtime device selection.

## Known Gaps

- No unit tests for coordinate mapping, display parsing, or recovery state machines.
- No automated native DLL or GPU test harness.
- No cross-platform CI; project is Windows-only today.
- `input motionevent` and persistent shell behavior vary by Android build.
