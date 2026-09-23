package main

import "testing"

func TestMapClientToVideoFitRejectsBars(t *testing.T) {
	if _, _, ok := mapClientToVideo(0, 10, 1024, 768, 1920, 1080, false); ok {
		t.Fatal("expected left edge to map outside fit viewport")
	}
	x, y, ok := mapClientToVideo(512, 384, 1024, 768, 1920, 1080, false)
	if !ok || x != 960 || y != 540 {
		t.Fatalf("center mapping = %d,%d,%v", x, y, ok)
	}
}

func TestMapClientToVideoFillCrops(t *testing.T) {
	x, y, ok := mapClientToVideo(0, 0, 1280, 720, 1600, 2560, true)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("fill mapping = %d,%d,%v", x, y, ok)
	}
}
