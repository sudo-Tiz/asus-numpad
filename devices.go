package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func scanDevices() (*Device, error) {
	file, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	device := &Device{}
	scanner := bufio.NewScanner(file)
	eventRe := regexp.MustCompile(`event(\d+)`)
	i2cRe := regexp.MustCompile(`i2c-(\d+)/`)
	
	var isTouchpad, isKeyboard bool

	for scanner.Scan() {
		line := scanner.Text()

		if (strings.Contains(line, `Name="ASUE`) || strings.Contains(line, `Name="ELAN`)) && 
			strings.Contains(line, "Touchpad") {
			isTouchpad, isKeyboard = true, false
		}

		if strings.Contains(line, `Name="AT Translated Set 2 keyboard`) || 
			strings.Contains(line, `Name="Asus Keyboard`) {
			isKeyboard, isTouchpad = true, false
		}

		if isTouchpad {
			if strings.HasPrefix(line, "S: ") {
				if m := i2cRe.FindStringSubmatch(line); len(m) > 1 {
					device.I2CDeviceID = m[1]
				}
			}
			if strings.HasPrefix(line, "H: ") {
				if m := eventRe.FindStringSubmatch(line); len(m) > 1 {
					device.TouchpadPath = "/dev/input/event" + m[1]
					isTouchpad = false
				}
			}
		}

		if isKeyboard && strings.HasPrefix(line, "H: ") {
			if m := eventRe.FindStringSubmatch(line); len(m) > 1 {
				device.KeyboardPath = "/dev/input/event" + m[1]
				isKeyboard = false
			}
		}

		if device.TouchpadPath != "" && device.KeyboardPath != "" && device.I2CDeviceID != "" {
			break
		}
	}

	if device.TouchpadPath == "" || device.KeyboardPath == "" || device.I2CDeviceID == "" {
		return nil, fmt.Errorf("device detection failed")
	}

	return device, scanner.Err()
}
