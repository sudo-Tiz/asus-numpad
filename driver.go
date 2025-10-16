package main

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"github.com/bendahl/uinput"
)

// Linux input event structure
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// Event types and codes
const (
	EV_ABS = 0x03
	EV_KEY = 0x01

	ABS_MT_POSITION_X = 0x35
	ABS_MT_POSITION_Y = 0x36
	BTN_TOOL_FINGER   = 0x145

	EVIOCGRAB = 0x40044590
)

// Driver manages the touchpad numpad functionality
type Driver struct {
	device        *Device
	layout        *Layout
	touchpadFd    int
	keyboard      uinput.Keyboard
	numlock       bool
	buttonPressed int
	x, y          int32
	minX, maxX    int32
	minY, maxY    int32
}

// NewDriver creates a new driver instance
func NewDriver(device *Device, layout *Layout) (*Driver, error) {
	d := &Driver{
		device:     device,
		layout:     layout,
		touchpadFd: -1,
	}

	// Open touchpad
	fd, err := syscall.Open(device.TouchpadPath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open touchpad: %w", err)
	}
	d.touchpadFd = fd

	// Get touchpad bounds
	if err := d.getTouchpadBounds(); err != nil {
		syscall.Close(d.touchpadFd)
		return nil, fmt.Errorf("failed to get bounds: %w", err)
	}

	log.Printf("Touchpad bounds: x[%d-%d] y[%d-%d]", d.minX, d.maxX, d.minY, d.maxY)

	// Create virtual keyboard
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("Asus Numpad"))
	if err != nil {
		syscall.Close(d.touchpadFd)
		return nil, fmt.Errorf("failed to create keyboard: %w", err)
	}
	d.keyboard = keyboard

	log.Println("Driver initialized")
	return d, nil
}

// Close cleans up resources
func (d *Driver) Close() {
	if d.numlock {
		d.deactivateNumlock()
	}
	if d.touchpadFd >= 0 {
		syscall.Close(d.touchpadFd)
	}
	if d.keyboard != nil {
		d.keyboard.Close()
	}
}

// Run starts the event loop
func (d *Driver) Run() error {
	log.Println("Event loop started")
	
	for {
		var event inputEvent
		data := (*(*[unsafe.Sizeof(event)]byte)(unsafe.Pointer(&event)))[:]
		n, err := syscall.Read(d.touchpadFd, data)
		
		if err != nil {
			if err == syscall.EAGAIN {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}

		if n == int(unsafe.Sizeof(event)) {
			d.processEvent(&event)
		}
	}
}

// processEvent handles input events
func (d *Driver) processEvent(event *inputEvent) {
	switch event.Type {
	case EV_ABS:
		if event.Code == ABS_MT_POSITION_X {
			d.x = event.Value
		} else if event.Code == ABS_MT_POSITION_Y {
			d.y = event.Value
		}
	case EV_KEY:
		if event.Code == BTN_TOOL_FINGER {
			if event.Value == 0 {
				d.handleFingerUp()
			} else if event.Value == 1 && d.buttonPressed == 0 {
				d.handleFingerDown()
			}
		}
	}
}

// handleFingerDown processes touch
func (d *Driver) handleFingerDown() {
	xRatio := float64(d.x) / float64(d.maxX)
	yRatio := float64(d.y) / float64(d.maxY)

	if *debugMode {
		log.Printf("Touch at x=%d y=%d (ratio: %.2f, %.2f)", d.x, d.y, xRatio, yRatio)
	}

	// Toggle in top-right corner
	if xRatio > 0.95 && yRatio < 0.09 {
		d.numlock = !d.numlock
		if d.numlock {
			d.activateNumlock()
		} else {
			d.deactivateNumlock()
		}
		return
	}

	if !d.numlock {
		return
	}

	// Normalize coordinates to 0-1 range
	xNorm := float64(d.x-d.minX) / float64(d.maxX-d.minX)
	yNorm := float64(d.y-d.minY) / float64(d.maxY-d.minY)

	// Map position to key
	col := int(math.Floor(float64(d.layout.Cols) * xNorm))
	row := int(math.Floor(float64(d.layout.Rows)*yNorm - d.layout.TopOffset))

	log.Printf("Touch x=%d y=%d (norm: %.2f,%.2f) -> row=%d col=%d (max: %dx%d)", 
		d.x, d.y, xNorm, yNorm, row, col, d.layout.Rows, d.layout.Cols)

	if row < 0 || row >= d.layout.Rows || col < 0 || col >= d.layout.Cols {
		log.Printf("Position out of bounds!")
		return
	}

	keyName := d.layout.Keys[row][col]
	
	log.Printf("Mapped to key: %s at [%d,%d]", keyName, row, col)
	
	if keyName == "KEY_RESERVED" || keyName == "" {
		log.Printf("Reserved/empty key, skipping")
		return
	}

	keyCode := keyNameToCode(keyName)
	if keyCode == 0 {
		log.Printf("Unknown key code for %s", keyName)
		return
	}

	d.buttonPressed = keyCode
	if err := d.keyboard.KeyDown(keyCode); err != nil {
		log.Printf("Failed to press key: %v", err)
	}
}

// handleFingerUp processes release
func (d *Driver) handleFingerUp() {
	if d.buttonPressed != 0 {
		d.keyboard.KeyUp(d.buttonPressed)
		d.buttonPressed = 0
	}
}

// activateNumlock enables numpad
func (d *Driver) activateNumlock() {
	d.keyboard.KeyPress(uinput.KeyNumlock)
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd), EVIOCGRAB, 1)
	d.sendI2C("0x01")
	time.Sleep(100 * time.Millisecond)
	d.sendI2C("0x01")
	log.Println("Numpad ON")
}

// deactivateNumlock disables numpad
func (d *Driver) deactivateNumlock() {
	d.keyboard.KeyPress(uinput.KeyNumlock)
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd), EVIOCGRAB, 0)
	d.sendI2C("0x00")
	log.Println("Numpad OFF")
}

// sendI2C controls LED
func (d *Driver) sendI2C(val string) {
	exec.Command("i2ctransfer", "-f", "-y", d.device.I2CDeviceID,
		"w13@0x15", "0x05", "0x00", "0x3d", "0x03", "0x06", "0x00",
		"0x07", "0x00", "0x0d", "0x14", "0x03", val, "0xad").Run()
}

// getTouchpadBounds retrieves the actual touchpad dimensions
func (d *Driver) getTouchpadBounds() error {
	// Structure for abs info
	type absInfo struct {
		Value      int32
		Minimum    int32
		Maximum    int32
		Fuzz       int32
		Flat       int32
		Resolution int32
	}

	const EVIOCGABS = 0x80184540 // EVIOCGABS(ABS_X) = 0x80184540 + axis

	// Get X bounds
	var absX absInfo
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(d.touchpadFd),
		uintptr(EVIOCGABS+ABS_MT_POSITION_X),
		uintptr(unsafe.Pointer(&absX)))
	if errno != 0 {
		return fmt.Errorf("failed to get X bounds: %v", errno)
	}

	// Get Y bounds
	var absY absInfo
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(d.touchpadFd),
		uintptr(EVIOCGABS+ABS_MT_POSITION_Y),
		uintptr(unsafe.Pointer(&absY)))
	if errno != 0 {
		return fmt.Errorf("failed to get Y bounds: %v", errno)
	}

	d.minX = absX.Minimum
	d.maxX = absX.Maximum
	d.minY = absY.Minimum
	d.maxY = absY.Maximum

	return nil
}

// keyNameToCode maps key names to codes
func keyNameToCode(name string) int {
	keys := map[string]int{
		"KEY_KP0":        uinput.KeyKp0,
		"KEY_KP1":        uinput.KeyKp1,
		"KEY_KP2":        uinput.KeyKp2,
		"KEY_KP3":        uinput.KeyKp3,
		"KEY_KP4":        uinput.KeyKp4,
		"KEY_KP5":        uinput.KeyKp5,
		"KEY_KP6":        uinput.KeyKp6,
		"KEY_KP7":        uinput.KeyKp7,
		"KEY_KP8":        uinput.KeyKp8,
		"KEY_KP9":        uinput.KeyKp9,
		"KEY_KPDOT":      uinput.KeyKpdot,
		"KEY_KPENTER":    uinput.KeyKpenter,
		"KEY_KPPLUS":     uinput.KeyKpplus,
		"KEY_KPMINUS":    uinput.KeyKpminus,
		"KEY_KPASTERISK": uinput.KeyKpasterisk,
		"KEY_KPSLASH":    uinput.KeyKpslash,
		"KEY_KPEQUAL":    uinput.KeyKpequal,
		"KEY_BACKSPACE":  uinput.KeyBackspace,
		"KEY_CALC":       uinput.KeyCalc,
	}
	return keys[name]
}
