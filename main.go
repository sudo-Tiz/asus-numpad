package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultLayoutPath = "/etc/asus-numpad/layout.json"
)

// Layout represents the numpad layout configuration
type Layout struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	TryTimes    int        `json:"try_times"`
	TrySleepMs  int        `json:"try_sleep_ms"`
	Cols        int        `json:"cols"`
	Rows        int        `json:"rows"`
	TopOffset   float64    `json:"top_offset"`
	Keys        [][]string `json:"keys"`
}

// Device holds information about detected input devices
type Device struct {
	TouchpadPath string
	KeyboardPath string
	I2CDeviceID  string
}

var (
	layoutFile = flag.String("layout-file", defaultLayoutPath, "Path to layout JSON file")
	debugMode  = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	// Setup logging
	if !*debugMode {
		log.SetFlags(0)
	}

	log.Println("Starting Asus Touchpad Numpad driver...")

	// Load layout configuration
	layout, err := loadLayout(*layoutFile)
	if err != nil {
		log.Fatalf("Failed to load layout: %v", err)
	}
	log.Printf("Loaded layout: %s", layout.Name)

	// Detect devices
	device, err := detectDevices(layout.TryTimes, time.Duration(layout.TrySleepMs)*time.Millisecond)
	if err != nil {
		log.Fatalf("Failed to detect devices: %v", err)
	}
	log.Printf("Detected touchpad: %s", device.TouchpadPath)
	log.Printf("Detected keyboard: %s", device.KeyboardPath)
	log.Printf("Detected I2C device: %s", device.I2CDeviceID)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create driver instance
	driver, err := NewDriver(device, layout)
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close()

	// Start event loop in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- driver.Run()
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Driver error: %v", err)
		}
	}

	log.Println("Asus Touchpad Numpad driver stopped")
}

// loadLayout reads and parses the layout JSON file
func loadLayout(path string) (*Layout, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout file: %w", err)
	}

	var layout Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("failed to parse layout JSON: %w", err)
	}

	// Validate layout
	if layout.Cols <= 0 || layout.Rows <= 0 {
		return nil, fmt.Errorf("invalid layout dimensions: cols=%d, rows=%d", layout.Cols, layout.Rows)
	}

	if len(layout.Keys) != layout.Rows {
		return nil, fmt.Errorf("layout rows mismatch: expected %d, got %d", layout.Rows, len(layout.Keys))
	}

	for i, row := range layout.Keys {
		if len(row) != layout.Cols {
			return nil, fmt.Errorf("layout cols mismatch at row %d: expected %d, got %d", i, layout.Cols, len(row))
		}
	}

	return &layout, nil
}

// detectDevices attempts to find the touchpad, keyboard, and I2C device
func detectDevices(tryTimes int, trySleep time.Duration) (*Device, error) {
	var device *Device
	var err error

	for i := 0; i < tryTimes; i++ {
		device, err = scanDevices()
		if err == nil {
			return device, nil
		}

		if i < tryTimes-1 {
			time.Sleep(trySleep)
		}
	}

	return nil, fmt.Errorf("failed to detect devices after %d attempts: %w", tryTimes, err)
}


