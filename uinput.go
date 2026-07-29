package main

import (
	"syscall"
	"unsafe"
)

// Minimal /dev/uinput virtual keyboard, replacing the now-archived
// github.com/bendahl/uinput dependency. Only the subset of the uinput
// API used by this driver (key press/release) is implemented.

const (
	uiSetEvbit   = 0x40045564 // _IOW('U', 100, int)
	uiSetKeybit  = 0x40045565 // _IOW('U', 101, int)
	uiDevSetup   = 0x405c5503 // _IOW('U', 3, struct uinput_setup)
	uiDevCreate  = 0x5501     // _IO('U', 1)
	uiDevDestroy = 0x5502     // _IO('U', 2)

	busUsb = 0x03

	evSyn     = 0x00
	evKey     = 0x01
	synReport = 0

	uinputMaxNameSize = 80
)

// Linux key codes (linux/input-event-codes.h). These are stable kernel ABI
// values and won't change.
const (
	KeyBackspace  = 14
	KeyKpasterisk = 55
	KeyNumlock    = 69
	KeyKp7        = 71
	KeyKp8        = 72
	KeyKp9        = 73
	KeyKpminus    = 74
	KeyKp4        = 75
	KeyKp5        = 76
	KeyKp6        = 77
	KeyKpplus     = 78
	KeyKp1        = 79
	KeyKp2        = 80
	KeyKp3        = 81
	KeyKp0        = 82
	KeyKpdot      = 83
	KeyKpenter    = 96
	KeyKpslash    = 98
	KeyKpequal    = 117
)

type inputID struct {
	Bustype, Vendor, Product, Version uint16
}

type uinputSetup struct {
	ID           inputID
	Name         [uinputMaxNameSize]byte
	FFEffectsMax uint32
}

// Keyboard is a virtual keyboard device created via /dev/uinput.
type Keyboard struct {
	fd int
}

func ioctl(fd int, req, arg uintptr) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg); errno != 0 {
		return errno
	}
	return nil
}

// createKeyboard opens the uinput device at path and registers it as a
// virtual keyboard capable of emitting the given key codes.
func createKeyboard(path string, name []byte, keys []int) (*Keyboard, error) {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	if err := ioctl(fd, uiSetEvbit, uintptr(evKey)); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}

	for _, key := range keys {
		if err := ioctl(fd, uiSetKeybit, uintptr(key)); err != nil {
			_ = syscall.Close(fd)
			return nil, err
		}
	}

	var setup uinputSetup
	setup.ID = inputID{Bustype: busUsb, Vendor: 0x1234, Product: 0x5678, Version: 1}
	copy(setup.Name[:], name)

	if err := ioctl(fd, uiDevSetup, uintptr(unsafe.Pointer(&setup))); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}

	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}

	return &Keyboard{fd: fd}, nil
}

func (k *Keyboard) writeEvent(evType, code uint16, value int32) error {
	ev := inputEvent{Type: evType, Code: code, Value: value}
	data := (*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:]
	_, err := syscall.Write(k.fd, data)
	return err
}

func (k *Keyboard) sync() error {
	return k.writeEvent(evSyn, synReport, 0)
}

// KeyDown sends a key press event for the given key code.
func (k *Keyboard) KeyDown(code int) error {
	if err := k.writeEvent(evKey, uint16(code), 1); err != nil {
		return err
	}
	return k.sync()
}

// KeyUp sends a key release event for the given key code.
func (k *Keyboard) KeyUp(code int) error {
	if err := k.writeEvent(evKey, uint16(code), 0); err != nil {
		return err
	}
	return k.sync()
}

// Close destroys the virtual device and releases the file descriptor.
func (k *Keyboard) Close() error {
	destroyErr := ioctl(k.fd, uiDevDestroy, 0)
	closeErr := syscall.Close(k.fd)
	if destroyErr != nil {
		return destroyErr
	}
	return closeErr
}
