package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// scanDevices parses /proc/bus/input/devices to find touchpad and keyboard
func scanDevices() (*Device, error) {
	file, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/bus/input/devices: %w", err)
	}
	defer file.Close()

	device := &Device{}
	scanner := bufio.NewScanner(file)
	
	var (
		touchpadDetected = 0
		keyboardDetected = 0
		currentIsTargetTouchpad = false
		currentIsTargetKeyboard = false
	)

	eventRegex := regexp.MustCompile(`event(\d+)`)
	i2cRegex := regexp.MustCompile(`i2c-(\d+)/`)

	for scanner.Scan() {
		line := scanner.Text()

		// Detect touchpad
		if touchpadDetected == 0 && (strings.Contains(line, `Name="ASUE`) || strings.Contains(line, `Name="ELAN`)) && strings.Contains(line, "Touchpad") {
			touchpadDetected = 1
			currentIsTargetTouchpad = true
			currentIsTargetKeyboard = false
			if *debugMode {
				log.Printf("Detected touchpad: %s", line)
			}
		}

		// Detect keyboard
		if keyboardDetected == 0 && (strings.Contains(line, `Name="AT Translated Set 2 keyboard`) || strings.Contains(line, `Name="Asus Keyboard`)) {
			keyboardDetected = 1
			currentIsTargetKeyboard = true
			currentIsTargetTouchpad = false
			if *debugMode {
				log.Printf("Detected keyboard: %s", line)
			}
		}

		// Extract touchpad information
		if currentIsTargetTouchpad {
			if strings.HasPrefix(line, "S: ") {
				// Extract I2C device ID
				matches := i2cRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					device.I2CDeviceID = matches[1]
					if *debugMode {
						log.Printf("I2C device ID: %s", device.I2CDeviceID)
					}
				}
			}
			
			if strings.HasPrefix(line, "H: ") {
				// Extract event number
				matches := eventRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					device.TouchpadPath = fmt.Sprintf("/dev/input/event%s", matches[1])
					touchpadDetected = 2
					currentIsTargetTouchpad = false
					if *debugMode {
						log.Printf("Touchpad path: %s", device.TouchpadPath)
					}
				}
			}
		}

		// Extract keyboard information
		if currentIsTargetKeyboard {
			if strings.HasPrefix(line, "H: ") {
				// Extract event number
				matches := eventRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					device.KeyboardPath = fmt.Sprintf("/dev/input/event%s", matches[1])
					keyboardDetected = 2
					currentIsTargetKeyboard = false
					if *debugMode {
						log.Printf("Keyboard path: %s", device.KeyboardPath)
					}
				}
			}
		}

		// Break early if all devices found
		if touchpadDetected == 2 && keyboardDetected == 2 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /proc/bus/input/devices: %w", err)
	}

	// Validate results
	if device.TouchpadPath == "" {
		return nil, fmt.Errorf("could not find touchpad")
	}
	if device.KeyboardPath == "" {
		return nil, fmt.Errorf("could not find keyboard")
	}
	if device.I2CDeviceID == "" {
		return nil, fmt.Errorf("could not find I2C device ID")
	}

	return device, nil
}
