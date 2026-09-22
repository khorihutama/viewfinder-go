//go:build windows

package android

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
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
	if x < 0 || y < 0 {
		return fmt.Errorf("tap coordinates must be non-negative")
	}
	_, err := d.Shell("input", "tap", fmt.Sprint(x), fmt.Sprint(y))
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
