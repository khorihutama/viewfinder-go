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
