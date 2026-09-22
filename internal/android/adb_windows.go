//go:build windows

package android

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Device struct {
	Serial string
	State  string
}

func Watch(done <-chan struct{}, changes chan<- []Device) {
	var previous string
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		devices, err := Devices()
		if err == nil {
			state := deviceState(devices)
			if state != previous {
				previous = state
				select {
				case changes <- devices:
				case <-done:
					return
				}
			}
		}
		select {
		case <-done:
			return
		case <-ticker.C:
		}
	}
}

func deviceState(devices []Device) string {
	var parts []string
	for _, device := range devices {
		parts = append(parts, device.Serial+":"+device.State)
	}
	return strings.Join(parts, ",")
}

func Devices() ([]Device, error) {
	cmd := exec.Command("adb", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("adb devices: %w", err)
	}

	var devices []Device
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] != "List" {
			devices = append(devices, Device{Serial: fields[0], State: fields[1]})
		}
	}
	return devices, scanner.Err()
}

func (d Device) Shell(args ...string) ([]byte, error) {
	if d.Serial == "" {
		return nil, fmt.Errorf("device serial is empty")
	}
	commandArgs := append([]string{"-s", d.Serial, "shell"}, args...)
	output, err := exec.Command("adb", commandArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("adb shell: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (d Device) Tap(x, y int) error {
	return d.TapDisplay(DisplayID(), x, y)
}

func DisplayID() int {
	if value, err := strconv.Atoi(os.Getenv("VIEWFINDER_ANDROID_DISPLAY_ID")); err == nil && value >= 0 {
		return value
	}
	return 7
}

func (d Device) TapDisplay(displayID, x, y int) error {
	if displayID < 0 {
		return fmt.Errorf("display ID must be non-negative")
	}
	if x < 0 || y < 0 {
		return fmt.Errorf("tap coordinates must be non-negative")
	}
	_, err := d.Shell("input", "-d", fmt.Sprint(displayID), "tap", fmt.Sprint(x), fmt.Sprint(y))
	return err
}

func (d Device) Swipe(x1, y1, x2, y2, durationMS int) error {
	if x1 < 0 || y1 < 0 || x2 < 0 || y2 < 0 || durationMS < 0 {
		return fmt.Errorf("swipe coordinates and duration must be non-negative")
	}
	_, err := d.Shell("input", "-d", fmt.Sprint(DisplayID()), "swipe", fmt.Sprint(x1), fmt.Sprint(y1), fmt.Sprint(x2), fmt.Sprint(y2), fmt.Sprint(durationMS))
	return err
}

func (d Device) Scroll(x, y, delta int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("scroll coordinates must be non-negative")
	}
	_, err := d.Shell("input", "-d", fmt.Sprint(DisplayID()), "swipe", fmt.Sprint(x), fmt.Sprint(y), fmt.Sprint(x), fmt.Sprint(y-delta), "300")
	return err
}

func FirstReady(devices []Device) (Device, bool) {
	for _, device := range devices {
		if device.State == "device" {
			return device, true
		}
	}
	return Device{}, false
}
