package display

import "testing"

func TestNormalizeDevicePathRejectsEmpty(t *testing.T) {
	if _, err := normalizeDevicePath(""); err == nil {
		t.Fatal("expected error for empty device path")
	}
}

func TestBuildOpenCandidatesIncludesExplicitPath(t *testing.T) {
	got := buildOpenCandidates("/dev/cu.usbmodemUSB35INCHIPSV21")
	if len(got) == 0 || got[0] != "/dev/cu.usbmodemUSB35INCHIPSV21" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestBuildOpenCandidatesIncludesTTYVariant(t *testing.T) {
	got := buildOpenCandidates("/dev/cu.usbmodemUSB35INCHIPSV21")
	if len(got) < 2 || got[1] != "/dev/tty.usbmodemUSB35INCHIPSV21" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}
