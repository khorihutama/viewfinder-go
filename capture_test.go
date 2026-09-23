package main

import (
	"testing"

	"github.com/khorihutama/viewfinder-go/internal/native"
)

func TestSelectCaptureDevice(t *testing.T) {
	devices := []native.CaptureDevice{{Index: 2, Name: "Card A"}, {Index: 5, Name: "Card B"}}
	if index, ok := selectCaptureDevice(devices, ""); !ok || index != 2 {
		t.Fatalf("default = %d, %v", index, ok)
	}
	if index, ok := selectCaptureDevice(devices, "5"); !ok || index != 5 {
		t.Fatalf("index = %d, %v", index, ok)
	}
	if index, ok := selectCaptureDevice(devices, "Card B"); !ok || index != 5 {
		t.Fatalf("name = %d, %v", index, ok)
	}
	if _, ok := selectCaptureDevice(devices, "missing"); ok {
		t.Fatal("missing device should fail")
	}
}
