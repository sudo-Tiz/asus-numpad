#!/usr/bin/env python3

import importlib
import logging
import math
import os
import re
import signal
import subprocess
import sys
from fcntl import F_SETFL, fcntl
from time import sleep
from typing import Optional, Tuple

from libevdev import EV_ABS, EV_KEY, EV_SYN, Device, InputEvent

# Setup logging
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
log = logging.getLogger('Pad')
log.setLevel(os.environ.get('LOG', 'INFO'))

# I2C command components for touchpad lighting control
I2C_CMD_PREFIX = ["i2ctransfer", "-f", "-y"]
I2C_CMD_DATA = ["w13@0x15", "0x05", "0x00", "0x3d", "0x03", "0x06", "0x00", "0x07", "0x00", "0x0d", "0x14", "0x03"]
I2C_CMD_SUFFIX = ["0xad"]
BRIGHT_VAL = "0x01"  # Maximum brightness for best visibility

# Global variables
model = 'm433ia'  # Default model
numlock = False   # Current numpad activation state
button_pressed = None  # Currently pressed numpad key

# Device handles
touchpad = None   # Touchpad device ID
keyboard = None   # Keyboard device ID
device_id = None  # I2C device ID
fd_t = None       # Touchpad file descriptor
fd_k = None       # Keyboard file descriptor
d_t = None        # Touchpad device object
d_k = None        # Keyboard device object
udev = None       # Virtual keyboard device
dev = None        # Virtual keyboard configuration

# Special keys
percentage_key = None  # Key used for percentage symbol
calculator_key = None  # Key used to launch calculator

# Current touch position
x = 0             # Current X position of touch
y = 0             # Current Y position of touch

# Touchpad boundaries
minx = 0          # Minimum X coordinate
maxx = 0          # Maximum X coordinate
miny = 0          # Minimum Y coordinate
maxy = 0          # Maximum Y coordinate

def cleanup(signum=None, frame=None):
    """Clean up resources before exiting"""
    log.info("Stopping touchpad numpad driver...")
    try:
        if 'numlock' in globals() and numlock:
            deactivate_numlock()
        if 'fd_t' in globals():
            fd_t.close()
        if 'fd_k' in globals():
            fd_k.close()
    except Exception as ex:
        log.error(f"Error during cleanup: {ex}")
    log.info("Exiting")
    sys.exit(0)



def detect_devices() -> Tuple[Optional[str], Optional[str], Optional[str], bool]:
    """Detect touchpad and keyboard devices from the system
    
    Returns:
        Tuple containing touchpad ID, keyboard ID, I2C device ID, and a success flag
    """
    keyboard_detected = 0
    touchpad_detected = 0
    touchpad_id = None
    keyboard_id = None
    device_i2c_id = None
    
    with open('/proc/bus/input/devices', 'r') as f:
        lines = f.readlines()
        for line in lines:
            # Detect ASUS/ELAN touchpad
            if touchpad_detected == 0 and ("Name=\"ASUE" in line or "Name=\"ELAN" in line) and "Touchpad" in line:
                touchpad_detected = 1
                log.debug('Detect touchpad from %s', line.strip())

            if touchpad_detected == 1:
                if "S: " in line:
                    device_i2c_id = re.sub(r".*i2c-(\d+)/.*$", r'\1', line).replace("\n", "")
                    log.debug('Set touchpad device id %s from %s', device_i2c_id, line.strip())

                if "H: " in line:
                    touchpad_id = line.split("event")[1].split(" ")[0]
                    touchpad_detected = 2
                    log.debug('Set touchpad id %s from %s', touchpad_id, line.strip())

            # Detect standard or ASUS keyboard
            if keyboard_detected == 0 and ("Name=\"AT Translated Set 2 keyboard" in line or "Name=\"Asus Keyboard" in line):
                keyboard_detected = 1
                log.debug('Detect keyboard from %s', line.strip())

            if keyboard_detected == 1:
                if "H: " in line:
                    keyboard_id = line.split("event")[1].split(" ")[0]
                    keyboard_detected = 2
                    log.debug('Set keyboard %s from %s', keyboard_id, line.strip())

            # Exit early if all devices found
            if keyboard_detected == 2 and touchpad_detected == 2:
                break
    
    return touchpad_id, keyboard_id, device_i2c_id, (keyboard_detected == 2 and touchpad_detected == 2)

def send_key_event(key: EV_KEY, is_pressed: bool = True) -> bool:
    """Send a key event (press or release) to the uinput device
    
    Args:
        key: The keyboard key to send
        is_pressed: True for key press, False for key release
        
    Returns:
        Success or failure status
    """
    try:
        event = InputEvent(key, 1 if is_pressed else 0)
        sync_event = InputEvent(EV_SYN.SYN_REPORT, 0)
        udev.send_events([event, sync_event])
        return True
    except OSError as ex:
        log.error(f"Failed to send key event: {ex}")
        return False

def send_i2c_command(brightness_value: str) -> bool:
    """Send an I2C command to the touchpad to control numpad lighting
    
    Args:
        brightness_value: Brightness level for the numpad (0x00=off, 0x01=on)
        
    Returns:
        Success or failure status
    """
    cmd_parts = I2C_CMD_PREFIX + [device_id] + I2C_CMD_DATA + [brightness_value] + I2C_CMD_SUFFIX
    try:
        subprocess.call(cmd_parts)
        return True
    except Exception as ex:
        log.error(f"Failed to send I2C command: {ex}")
        return False

def activate_numlock() -> None:
    """Activate the numpad functionality"""
    try:
        # Send numlock key press
        send_key_event(EV_KEY.KEY_NUMLOCK, True)
        # Grab the touchpad to prevent normal touchpad events
        d_t.grab()
        
        # Turn on the numpad lights
        send_i2c_command(BRIGHT_VAL)
        # Second call for hardware reliability
        sleep(0.1)
        send_i2c_command(BRIGHT_VAL)
        
        log.debug("Numpad activated")
    except Exception as ex:
        log.error(f"Failed to activate numlock: {ex}")

def deactivate_numlock() -> None:
    """Deactivate the numpad functionality"""
    try:
        # Send numlock key release
        send_key_event(EV_KEY.KEY_NUMLOCK, False)
        # Release the touchpad to restore normal touchpad functionality
        d_t.ungrab()
        
        # Turn off the numpad lights
        send_i2c_command("0x00")
        
        log.debug("Numpad deactivated")
    except Exception as ex:
        log.error(f"Failed to deactivate numlock: {ex}")

def launch_calculator() -> None:
    """Simulate calculator key press and release"""
    try:
        # Press calculator key
        send_key_event(calculator_key, True)
        # Release calculator key
        send_key_event(calculator_key, False)
        log.debug("Calculator launched")
    except Exception as ex:
        log.error(f"Failed to launch calculator: {ex}")



def initialize_model():
    """Initialize the model and load keyboard layout"""
    global model, model_layout
    
    # Select model from command line if provided
    if len(sys.argv) > 1:
        model = sys.argv[1]
    
    # Check if model exists before importing
    try:
        available_models = [f[:-3] for f in os.listdir('numpad_layouts') if f.endswith('.py') and not f.startswith('__')]
        if model not in available_models:
            log.error(f"Model '{model}' not available. Options: {', '.join(available_models)}")
            sys.exit(1)
        model_layout = importlib.import_module('numpad_layouts.' + model)
        return model_layout
    except ImportError:
        log.error(f"Failed to import layout module for model '{model}'")
        sys.exit(1)
    except FileNotFoundError:
        log.error("Could not find numpad_layouts directory")
        sys.exit(1)

def setup_input_devices():
    """Setup input devices (touchpad and keyboard) based on detected hardware"""
    global touchpad, keyboard, device_id, fd_t, fd_k, d_t, d_k, minx, maxx, miny, maxy
    
    # Detect input devices with retries
    attempts_remaining = model_layout.try_times
    while attempts_remaining > 0:
        touchpad, keyboard, device_id, devices_found = detect_devices()
        
        if devices_found:
            break
        
        attempts_remaining -= 1
        if attempts_remaining == 0:
            if not keyboard:
                log.error("Can't find keyboard")
            if not touchpad:
                log.error("Can't find touchpad")
            if touchpad and not device_id or (device_id and not device_id.isnumeric()):
                log.error("Can't find touchpad I2C device id")
            sys.exit(1)
        
        sleep(model_layout.try_sleep)

    # Start monitoring the touchpad
    try:
        fd_t = open('/dev/input/event' + str(touchpad), 'rb')
        fcntl(fd_t, F_SETFL, os.O_NONBLOCK)
        d_t = Device(fd_t)
    except (IOError, OSError) as ex:
        log.error(f"Could not open touchpad device: {ex}")
        sys.exit(1)

    # Retrieve touchpad dimensions
    try:
        ai = d_t.absinfo[EV_ABS.ABS_X]
        (minx, maxx) = (ai.minimum, ai.maximum)
        ai = d_t.absinfo[EV_ABS.ABS_Y]
        (miny, maxy) = (ai.minimum, ai.maximum)
        log.debug('Touchpad min-max: x %d-%d, y %d-%d', minx, maxx, miny, maxy)
    except Exception as ex:
        log.error(f"Failed to get touchpad dimensions: {ex}")
        fd_t.close()
        sys.exit(1)

    # Start monitoring the keyboard (numlock)
    try:
        fd_k = open('/dev/input/event' + str(keyboard), 'rb')
        fcntl(fd_k, F_SETFL, os.O_NONBLOCK)
        d_k = Device(fd_k)
    except (IOError, OSError) as ex:
        log.error(f"Could not open keyboard device: {ex}")
        fd_t.close()
        sys.exit(1)

def setup_virtual_keyboard():
    """Set up the virtual keyboard device for sending numpad events"""
    global udev, dev, percentage_key, calculator_key
    
    # Set up percentage key based on keyboard layout
    percentage_key = EV_KEY.KEY_5  # Default (QWERTY layout)
    calculator_key = EV_KEY.KEY_CALC

    if len(sys.argv) > 2:
        try:
            percentage_key = EV_KEY.codes[int(sys.argv[2])]
        except (IndexError, ValueError) as ex:
            log.error(f"Invalid percentage key code: {ex}")
            sys.exit(1)

    # Create a new keyboard device to send numpad events
    try:
        dev = Device()
        dev.name = "Asus Touchpad/Numpad"
        dev.enable(EV_KEY.KEY_LEFTSHIFT)
        dev.enable(EV_KEY.KEY_NUMLOCK)
        dev.enable(calculator_key)

        for col in model_layout.keys:
            for key in col:
                dev.enable(key)

        if percentage_key != EV_KEY.KEY_5:
            dev.enable(percentage_key)

        udev = dev.create_uinput_device()
    except Exception as ex:
        log.error(f"Failed to create virtual input device: {ex}")
        fd_t.close()
        fd_k.close()
        sys.exit(1)

def process_events():
    """Process touchpad events and generate numpad key events"""
    global numlock, button_pressed, x, y
    
    log.info("Asus Touchpad Numpad driver started successfully")
    
    try:
        while True:
            for touch_event in d_t.events():
                # Filter for position and finger events only
                if not (
                    touch_event.matches(EV_ABS.ABS_MT_POSITION_X) or
                    touch_event.matches(EV_ABS.ABS_MT_POSITION_Y) or
                    touch_event.matches(EV_KEY.BTN_TOOL_FINGER)
                ):
                    continue

                # Track touch position
                if touch_event.matches(EV_ABS.ABS_MT_POSITION_X):
                    x = touch_event.value
                    continue
                if touch_event.matches(EV_ABS.ABS_MT_POSITION_Y):
                    y = touch_event.value
                    continue

                # Handle finger up event
                if touch_event.value == 0:
                    log.debug('finger up at x %d y %d', x, y)
                    if button_pressed:
                        log.debug('send key up event %s', button_pressed)
                        if button_pressed == percentage_key:
                            send_key_event(EV_KEY.KEY_LEFTSHIFT, False)
                        send_key_event(button_pressed, False)
                        button_pressed = None

                # Handle finger down event
                elif touch_event.value == 1 and not button_pressed:
                    log.debug('finger down at x %d y %d', x, y)

                    # Top-right corner: toggle numlock
                    if (x > 0.95 * maxx) and (y < 0.09 * maxy):
                        numlock = not numlock
                        if numlock:
                            activate_numlock()
                        else:
                            deactivate_numlock()
                        continue

                    # Top-left corner: calculator shortcut
                    elif (x < 0.06 * maxx) and (y < 0.07 * maxy):
                        if not numlock:
                            launch_calculator()
                        continue

                    # Only process numpad presses when numlock is active
                    if not numlock:
                        continue

                    # Map touch position to numpad key
                    col = math.floor(model_layout.cols * x / (maxx+1))
                    row = math.floor((model_layout.rows * y / maxy) - model_layout.top_offset)
                    
                    if row < 0:
                        continue
                        
                    try:
                        button_pressed = model_layout.keys[row][col]
                    except IndexError:
                        log.debug('Unhandled col/row %d/%d for position %d-%d', col, row, x, y)
                        continue
                    
                    # Handle percentage symbol (requires shift)
                    if button_pressed == EV_KEY.KEY_5:
                        button_pressed = percentage_key

                    log.debug('send press key event %s', button_pressed)
                    if button_pressed == percentage_key:
                        send_key_event(EV_KEY.KEY_LEFTSHIFT, True)
                    send_key_event(button_pressed, True)

            sleep(0.1)  # Reduce CPU usage
            
    except KeyboardInterrupt:
        # Handle Ctrl+C
        pass
    except Exception as ex:
        log.error(f"Unexpected error: {ex}")
    finally:
        # Clean up resources
        cleanup()

if __name__ == "__main__":
    # Register signal handlers
    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)
    
    # Initialize driver components
    model_layout = initialize_model()
    setup_input_devices()
    setup_virtual_keyboard()
    
    # Start event processing
    process_events()
