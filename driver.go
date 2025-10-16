package main

import (
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"github.com/bendahl/uinput"
)

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

const (
	EV_ABS            = 0x03
	EV_KEY            = 0x01
	ABS_MT_POSITION_X = 0x35
	ABS_MT_POSITION_Y = 0x36
	BTN_TOOL_FINGER   = 0x145
	EVIOCGRAB         = 0x40044590
	EVIOCGABS         = 0x80184540
)

var keyCodeMap = map[string]int{
	"KEY_KP0": uinput.KeyKp0, "KEY_KP1": uinput.KeyKp1, "KEY_KP2": uinput.KeyKp2,
	"KEY_KP3": uinput.KeyKp3, "KEY_KP4": uinput.KeyKp4, "KEY_KP5": uinput.KeyKp5,
	"KEY_KP6": uinput.KeyKp6, "KEY_KP7": uinput.KeyKp7, "KEY_KP8": uinput.KeyKp8,
	"KEY_KP9": uinput.KeyKp9, "KEY_KPDOT": uinput.KeyKpdot, "KEY_KPENTER": uinput.KeyKpenter,
	"KEY_KPPLUS": uinput.KeyKpplus, "KEY_KPMINUS": uinput.KeyKpminus,
	"KEY_KPASTERISK": uinput.KeyKpasterisk, "KEY_KPSLASH": uinput.KeyKpslash,
	"KEY_KPEQUAL": uinput.KeyKpequal, "KEY_BACKSPACE": uinput.KeyBackspace,
}

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
	cols, rows    float64
}

func NewDriver(device *Device, layout *Layout) (*Driver, error) {
	fd, err := syscall.Open(device.TouchpadPath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	d := &Driver{
		device:     device,
		layout:     layout,
		touchpadFd: fd,
		cols:       float64(layout.Cols),
		rows:       float64(layout.Rows),
	}

	if err := d.getTouchpadBounds(); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("Asus Numpad"))
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}
	d.keyboard = keyboard

	return d, nil
}

func (d *Driver) Close() {
	if d.numlock {
		d.deactivateNumlock()
	}
	syscall.Close(d.touchpadFd)
	d.keyboard.Close()
}

func (d *Driver) Run() error {
	for {
		var event inputEvent
		data := (*(*[unsafe.Sizeof(event)]byte)(unsafe.Pointer(&event)))[:]
		
		if n, err := syscall.Read(d.touchpadFd, data); err != nil {
			if err == syscall.EAGAIN {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		} else if n == int(unsafe.Sizeof(event)) {
			d.processEvent(&event)
		}
	}
}

func (d *Driver) processEvent(event *inputEvent) {
	if event.Type == EV_ABS {
		if event.Code == ABS_MT_POSITION_X {
			d.x = event.Value
		} else if event.Code == ABS_MT_POSITION_Y {
			d.y = event.Value
		}
	} else if event.Type == EV_KEY && event.Code == BTN_TOOL_FINGER {
		if event.Value == 0 {
			d.handleFingerUp()
		} else if event.Value == 1 && d.buttonPressed == 0 {
			d.handleFingerDown()
		}
	}
}

func (d *Driver) handleFingerDown() {
	xRatio := float64(d.x) / float64(d.maxX)
	yRatio := float64(d.y) / float64(d.maxY)

	// Toggle numpad in top-right corner
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

	// Map touch to key position
	xNorm := float64(d.x-d.minX) / float64(d.maxX-d.minX)
	yNorm := float64(d.y-d.minY) / float64(d.maxY-d.minY)
	col := int(d.cols * xNorm)
	row := int(d.rows*yNorm - d.layout.TopOffset)

	if row < 0 || row >= d.layout.Rows || col < 0 || col >= d.layout.Cols {
		return
	}

	keyName := d.layout.Keys[row][col]
	if keyName == "KEY_RESERVED" {
		return
	}

	if keyCode := keyNameToCode(keyName); keyCode != 0 {
		d.buttonPressed = keyCode
		d.keyboard.KeyDown(keyCode)
	}
}

func (d *Driver) handleFingerUp() {
	if d.buttonPressed != 0 {
		d.keyboard.KeyUp(d.buttonPressed)
		d.buttonPressed = 0
	}
}

func (d *Driver) activateNumlock() {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd), EVIOCGRAB, 1)
	d.sendI2C("0x01")
}

func (d *Driver) deactivateNumlock() {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd), EVIOCGRAB, 0)
	d.sendI2C("0x00")
}

func (d *Driver) sendI2C(val string) {
	exec.Command("i2ctransfer", "-f", "-y", d.device.I2CDeviceID,
		"w13@0x15", "0x05", "0x00", "0x3d", "0x03", "0x06", "0x00",
		"0x07", "0x00", "0x0d", "0x14", "0x03", val, "0xad").Run()
}

func (d *Driver) getTouchpadBounds() error {
	type absInfo struct {
		Value, Minimum, Maximum, Fuzz, Flat, Resolution int32
	}

	var absX, absY absInfo
	
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd),
		EVIOCGABS+ABS_MT_POSITION_X, uintptr(unsafe.Pointer(&absX))); errno != 0 {
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.touchpadFd),
		EVIOCGABS+ABS_MT_POSITION_Y, uintptr(unsafe.Pointer(&absY))); errno != 0 {
		return errno
	}

	d.minX, d.maxX = absX.Minimum, absX.Maximum
	d.minY, d.maxY = absY.Minimum, absY.Maximum
	return nil
}

func keyNameToCode(name string) int {
	return keyCodeMap[name]
}
