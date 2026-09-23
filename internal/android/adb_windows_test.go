//go:build windows

package android

import "testing"

func TestSelectReady(t *testing.T) {
	devices := []Device{{Serial: "a", State: "offline"}, {Serial: "b", State: "device"}}
	if selected, ok := SelectReady(devices, "b"); !ok || selected.Serial != "b" {
		t.Fatalf("selected = %#v, %v", selected, ok)
	}
	if _, ok := SelectReady(devices, "missing"); ok {
		t.Fatal("missing selected device should fail")
	}
}
