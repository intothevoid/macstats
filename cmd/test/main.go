package main

import (
	"fmt"
	"log"
	"os"

	"macstats/internal/display"
)

func main() {
	dataFile := "resources/themes/3.5inchTheme1.data"
	data, err := os.ReadFile(dataFile)
	if err != nil {
		log.Fatal(err)
	}

	dev, err := resolveDevice()
	if err != nil {
		log.Fatal(err)
	}

	writer, err := display.NewWriter(dev)
	if err != nil {
		log.Fatal(err)
	}

	if err := writer.WriteFrame(data); err != nil {
		log.Fatal(err)
	}
}

func resolveDevice() (string, error) {
	if device := os.Getenv("DISPLAY_DEVICE"); device != "" {
		return device, nil
	}

	devices, err := display.DetectUSBSerialDevices()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("no USB serial display devices detected; set DISPLAY_DEVICE to override")
	}

	return devices[0], nil
}
