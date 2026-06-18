package protocol

import (
	"bytes"
	"os"
	"testing"
)

func TestParseThemeSeparatesDotNetHeaderFromPayload(t *testing.T) {
	data, err := os.ReadFile("../../resources/themes/3.5inchTheme1.data")
	if err != nil {
		t.Fatal(err)
	}

	fixture, err := os.ReadFile("../../resources/testdata/theme_3_5inch_header.bin")
	if err != nil {
		t.Fatal(err)
	}

	theme, err := ParseTheme(data)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(theme.Raw, data) {
		t.Fatal("expected raw theme bytes to be preserved")
	}

	if theme.HeaderLength != 256 {
		t.Fatalf("got header length %d", theme.HeaderLength)
	}

	if !bytes.Equal(theme.Header, fixture[:theme.HeaderLength]) {
		t.Fatal("expected header to match curated fixture bytes")
	}

	if !bytes.Equal(theme.Payload[:len(fixture)-theme.HeaderLength], fixture[theme.HeaderLength:]) {
		t.Fatal("expected payload prefix to follow header bytes")
	}

	if len(theme.Payload) == 0 {
		t.Fatal("expected payload bytes")
	}
}

func TestParseThemeRejectsShortBuffers(t *testing.T) {
	if _, err := ParseTheme([]byte{0x01, 0x02}); err == nil {
		t.Fatal("expected short buffer error")
	}
}

func TestThemeStartupPayloadReturnsRawBytes(t *testing.T) {
	theme := Theme{Raw: []byte{0x01, 0x02, 0x03}}

	if got := theme.StartupPayload(); len(got) != 3 {
		t.Fatalf("unexpected payload length: %d", len(got))
	}
}
