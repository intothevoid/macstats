package display

import (
	"os"
)

// Writer writes framebuffer data to a display.
// The protocol is currently unknown — it needs live testing.
// On successful writing, the display should update.
type Writer struct {
	device string  // e.g. "/dev/cu.usbmodemUSB35INCHIPSV21"
}

func NewWriter(device string) *Writer {
	return &Writer{device: device}
}

func (w *Writer) WriteFrame(buffer []byte) error {
	// Try to write to the device (O_RDWR | O_NOCTTY)
	fd, err := os.OpenFile(w.device, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.Write(buffer)
	return err
}

// CreateFrame 320x272 16-bit RGB565 frame with header (FB Protocol)
// 8 bytes header + pixel data
func CreateFrame(width, height uint16) []byte {
	n := 8 + int(width*height*2)
	buf := make([]byte, n)
	buf[0] = 0x13  // Frame buffer data command
	buf[1] = byte(width >> 8)  // Width MSB
	buf[2] = byte(width & 0xFF)  // Width LSB
	buf[3] = byte(height >> 8)  // Height MSB
	buf[4] = byte(height & 0xFF)  // Height LSB
	buf[5] = 0x10  // 16-bit depth (RGB565)
	buf[6] = 0x00  // Reserved
	buf[7] = 0x00  // Reserved
	return buf
}
