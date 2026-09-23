//go:build windows

package android

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Device struct {
	Serial  string
	State   string
	Display int
}

type InputSession struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	displayID int
	mu        sync.Mutex
}

func (s *InputSession) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stdin != nil
}

func (d Device) StartInput() (*InputSession, error) {
	displayID, err := d.targetDisplay()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("adb", "-s", d.Serial, "shell")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("start adb input session: %w", err)
	}
	return &InputSession{cmd: cmd, stdin: stdin, displayID: displayID}, nil
}

func (s *InputSession) command(displayID int, args ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdin == nil {
		return fmt.Errorf("input session is closed")
	}
	line := "input -d " + strconv.Itoa(displayID) + " " + strings.Join(args, " ") + "\n"
	_, err := io.WriteString(s.stdin, line)
	return err
}

func (s *InputSession) Tap(x, y int) error {
	return s.command(s.displayID, "tap", strconv.Itoa(x), strconv.Itoa(y))
}

func (s *InputSession) Swipe(x1, y1, x2, y2, durationMS int) error {
	return s.command(s.displayID, "swipe", strconv.Itoa(x1), strconv.Itoa(y1), strconv.Itoa(x2), strconv.Itoa(y2), strconv.Itoa(durationMS))
}

func (s *InputSession) Scroll(x, y, delta int) error {
	return s.Swipe(x, y, x, y-delta, 80)
}

func (s *InputSession) Motion(action string, x, y int) error {
	if action != "DOWN" && action != "MOVE" && action != "UP" {
		return fmt.Errorf("invalid motion action: %s", action)
	}
	return s.command(s.displayID, "motionevent", action, strconv.Itoa(x), strconv.Itoa(y))
}

func (s *InputSession) KeyEvent(key string) error {
	if key == "" {
		return fmt.Errorf("key name is empty")
	}
	return s.command(s.displayID, "keyevent", key)
}

func (s *InputSession) Close() error {
	s.mu.Lock()
	if s.stdin == nil {
		s.mu.Unlock()
		return nil
	}
	stdin := s.stdin
	s.stdin = nil
	s.mu.Unlock()
	stdin.Close()
	return s.cmd.Wait()
}

func ResolveDisplayID(d Device) (int, error) {
	output, err := d.Shell("dumpsys", "display")
	if err != nil {
		return 0, err
	}
	return parseDisplayID(string(output))
}

func parseDisplayID(output string) (int, error) {
	pending := -1
	external := false
	highest := 0
	for _, line := range strings.Split(output, "\n") {
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "HDMI") || strings.Contains(upper, "EXTERNAL") {
			external = true
			if pending > 0 {
				return pending, nil
			}
		}
		if index := strings.Index(line, "mDisplayId="); index >= 0 {
			value := strings.TrimSpace(line[index+len("mDisplayId="):])
			value = strings.TrimSpace(strings.TrimPrefix(value, ":"))
			if fields := strings.Fields(value); len(fields) > 0 {
				value = strings.Trim(fields[0], " ,}")
			}
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				pending = parsed
				if parsed > highest {
					highest = parsed
				}
				if external && pending > 0 {
					return pending, nil
				}
			}
		}
	}
	if highest > 0 {
		return highest, nil
	}
	return 0, fmt.Errorf("external HDMI display ID not found (display 0 is built-in)")
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
			devices = append(devices, Device{Serial: fields[0], State: fields[1], Display: -1})
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
	displayID := d.Display
	if displayID <= 0 {
		displayID = DisplayID()
	}
	if displayID <= 0 {
		return fmt.Errorf("external display ID is not configured")
	}
	return d.TapDisplay(displayID, x, y)
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

func (d Device) targetDisplay() (int, error) {
	if d.Display > 0 {
		return d.Display, nil
	}
	displayID := DisplayID()
	if displayID <= 0 {
		return 0, fmt.Errorf("external display ID is not configured")
	}
	return displayID, nil
}

func (d Device) Swipe(x1, y1, x2, y2, durationMS int) error {
	if x1 < 0 || y1 < 0 || x2 < 0 || y2 < 0 || durationMS < 0 {
		return fmt.Errorf("swipe coordinates and duration must be non-negative")
	}
	displayID, err := d.targetDisplay()
	if err != nil {
		return err
	}
	_, err = d.Shell("input", "-d", fmt.Sprint(displayID), "swipe", fmt.Sprint(x1), fmt.Sprint(y1), fmt.Sprint(x2), fmt.Sprint(y2), fmt.Sprint(durationMS))
	return err
}

func (d Device) Scroll(x, y, delta int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("scroll coordinates must be non-negative")
	}
	displayID, err := d.targetDisplay()
	if err != nil {
		return err
	}
	_, err = d.Shell("input", "-d", fmt.Sprint(displayID), "swipe", fmt.Sprint(x), fmt.Sprint(y), fmt.Sprint(x), fmt.Sprint(y-delta), "300")
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

func SelectReady(devices []Device, serial string) (Device, bool) {
	if serial != "" {
		for _, device := range devices {
			if device.Serial == serial && device.State == "device" {
				return device, true
			}
		}
		return Device{}, false
	}
	return FirstReady(devices)
}
