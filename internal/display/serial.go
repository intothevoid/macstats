package display

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type SerialWriter struct {
	devicePath string
}

func NewSerialWriter(devicePath string) (*SerialWriter, error) {
	path, err := normalizeDevicePath(devicePath)
	if err != nil {
		return nil, err
	}

	return &SerialWriter{devicePath: path}, nil
}

func normalizeDevicePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("device path is required")
	}

	return trimmed, nil
}

func buildOpenCandidates(path string) []string {
	candidates := []string{path}

	switch {
	case strings.Contains(path, "/cu."):
		candidates = append(candidates, strings.Replace(path, "/cu.", "/tty.", 1))
	case strings.Contains(path, "/tty."):
		candidates = append(candidates, strings.Replace(path, "/tty.", "/cu.", 1))
	}

	return candidates
}

func DetectUSBSerialDevices() ([]string, error) {
	return filepath.Glob("/dev/cu.usbmodem*")
}

func (w *SerialWriter) Write(payload []byte) error {
	var lastErr error
	for _, candidate := range buildOpenCandidates(w.devicePath) {
		fd, err := syscall.Open(candidate, os.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			lastErr = err
			continue
		}

		file := os.NewFile(uintptr(fd), candidate)
		_, err = file.Write(payload)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	}

	return lastErr
}
