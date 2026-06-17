package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	dev := "/dev/cu.usbmodemUSB35INCHIPSV21"
	dataFile := "/Users/bindok/project/macstats/35inchENG/config/3.5inchTheme1.data"
	data, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Skip 256-byte .NET serialization header
	payload := data[256:]
	
	if len(payload) < 4 {
		fmt.Println("Error: payload too short")
		return
	}
	
	// Print protocol marker
	marker := payload[:4]
	fmt.Printf("Protocol marker: %02x %02x %02x %02x\n", marker[0], marker[1], marker[2], marker[3])
	fmt.Printf("Pixel data (%d bytes)\n", len(payload)-4)
	
	// Now send to the device
	const (
		O_RDWR   = 2
		O_NOCTTY = 8
	)
	fd, err := syscall.Open(dev, O_RDWR|O_NOCTTY, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer syscall.Close(fd)
	
	// Try: send the FULL data file (including header)
	wrote, err := syscall.Write(fd, data)
	if err != nil {
		fmt.Printf("Full file write error: %v\n", err)
	} else {
		fmt.Printf("Full file: wrote %d bytes\n", wrote)
	}
	
	// Wait, then try payload only
	wrote, err = syscall.Write(fd, payload)
	if err != nil {
		fmt.Printf("Payload write error: %v\n", err)
	} else {
		fmt.Printf("Payload: wrote %d bytes\n", wrote)
	}
}
