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

const defaultLayoutPath = "/etc/asus-numpad/layout.json"

type Layout struct {
	TryTimes         int        `json:"try_times"`
	TrySleepMs       int        `json:"try_sleep_ms"`
	Cols             int        `json:"cols"`
	Rows             int        `json:"rows"`
	TopOffset        float64    `json:"top_offset"`
	NumlockIsEnabled bool       `json:"numlock_is_enabled"`
	Keys             [][]string `json:"keys"`
}

type Device struct {
	TouchpadPath string
	I2CDeviceID  string
}

var layoutFile = flag.String("layout-file", defaultLayoutPath, "Path to layout JSON")

func main() {
	flag.Parse()
	log.SetFlags(0)

	layout, err := loadLayout(*layoutFile)
	if err != nil {
		log.Fatal(err)
	}

	device, err := detectDevices(layout.TryTimes, time.Duration(layout.TrySleepMs)*time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}

	driver, err := NewDriver(device, layout)
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() { errChan <- driver.Run() }()

	select {
	case <-sigChan:
	case err := <-errChan:
		if err != nil {
			log.Fatal(err)
		}
	}
}

func loadLayout(path string) (*Layout, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var layout Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, err
	}

	if layout.Cols <= 0 || layout.Rows <= 0 || len(layout.Keys) != layout.Rows {
		return nil, fmt.Errorf("invalid layout")
	}

	return &layout, nil
}

func detectDevices(tryTimes int, trySleep time.Duration) (*Device, error) {
	for i := 0; i < tryTimes; i++ {
		if device, err := scanDevices(); err == nil {
			return device, nil
		}
		if i < tryTimes-1 {
			time.Sleep(trySleep)
		}
	}
	return nil, fmt.Errorf("device detection failed")
}
