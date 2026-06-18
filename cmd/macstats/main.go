package main

import (
	"fmt"
	"log"
	"os"

	"macstats/internal/display"
	"macstats/internal/protocol"
)

func main() {
	device, err := resolveDevice()
	if err != nil {
		log.Fatal(err)
	}

	data, err := os.ReadFile("resources/themes/3.5inchTheme1.data")
	if err != nil {
		log.Fatal(err)
	}

	theme, err := protocol.ParseTheme(data)
	if err != nil {
		log.Fatal(err)
	}

	writer, err := display.NewWriter(device)
	if err != nil {
		log.Fatal(err)
	}

	if err := writer.WriteFrame(theme.StartupPayload()); err != nil {
		log.Fatal(err)
	}
}

func resolveDevice() (string, error) {
	if device := os.Getenv("MACSTATS_DEVICE"); device != "" {
		return device, nil
	}

	devices, err := display.DetectUSBSerialDevices()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("no USB serial display devices detected; set MACSTATS_DEVICE to override")
	}

	return devices[0], nil
}
