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

func TestParseDisplayID(t *testing.T) {
	for _, output := range []string{
		"mDisplayId=0\nmDisplayId=8",
		"mDisplayId= 0\nmDisplayId=: 8",
	} {
		if id, err := parseDisplayID(output); err != nil || id != 8 {
			t.Fatalf("id = %d, err = %v", id, err)
		}
	}
	if _, err := parseDisplayID("mDisplayId=0"); err == nil {
		t.Fatal("built-in display should not resolve as external")
	}
}
