# Asus Touchpad Numpad Driver

A simple Linux driver to enable the numpad functionality on Asus laptops with touchpad-integrated numpads. This allows you to switch between normal touchpad usage and numpad mode with a simple touch.

## TLDR

```bash
git clone https://github.com/sudo-Tiz/asus-numpad.git && cd asus-numpad && sudo ./install.sh
```

## Features

- Toggle numpad mode by tapping the top-right corner of the touchpad
- Launch calculator by tapping the top-left corner of the touchpad
- Supports multiple Asus laptop models with different numpad layouts
- QWERTY and AZERTY keyboard layout support

## Supported Models

The driver automatically detects your touchpad and includes layouts for:

- `gx701` - Zephyrus S GX701
- `m433ia` - VivoBook M433IA and similar models with % and = symbols
- `ux433fa` - ZenBook UX433FA and similar models without extra symbols
- `ux581l` - ZenBook Pro Duo UX581L 

## Requirements

- Linux distribution with systemd
- i2c-tools package
- libevdev2 and python3-libevdev packages

## Installation

### 1. Install Dependencies

For Debian/Ubuntu-based distributions:
```bash
sudo apt install libevdev2 python3-libevdev i2c-tools git
```

For Arch-based distributions:
```bash
sudo pacman -S libevdev python-libevdev i2c-tools git
```

For Fedora:
```bash
sudo dnf install libevdev python-libevdev i2c-tools git
```

### 2. Install the Driver

Clone this repository and run the install script:
```bash
git clone https://github.com/sudo-Tiz/asus-numpad.git
cd asus-numpad
sudo ./install.sh
```

The installation script will:
1. Check for dependencies
2. Detect and test your touchpad
3. Ask you to select your laptop model
4. Ask for your keyboard layout (QWERTY or AZERTY)
5. Set up and enable the systemd service

## Usage

- **Toggle numpad mode**: Tap the top-right corner of your touchpad
- **Launch calculator**: Tap the top-left corner of your touchpad

## Troubleshooting

### Viewing Logs

To see the service logs:
```bash
journalctl -u asus_touchpad_numpad
```

For real-time log viewing:
```bash
journalctl -fu asus_touchpad_numpad
```

### Enable Debug Logging

To run the script with debug logging:
```bash
LOG=DEBUG sudo -E /usr/share/asus_touchpad_numpad-driver/asus_touchpad.py
```

### Boot Failure

If the service fails to start at boot (common on some distributions like Pop!_OS, Linux Mint, Elementary OS, or Solus OS), you can increase the sleep time in the service file:

```bash
sudo nano /etc/systemd/system/asus_touchpad_numpad.service
```

Adjust the ExecStartPre line to increase the delay:
```
ExecStartPre=/bin/sleep 5
```

### Uninstallation

To uninstall:
```bash
sudo ./uninstall.sh
```

## Adding New Layouts

To add support for a new laptop model, create a new Python file in the `numpad_layouts` directory with the appropriate key layout configuration.

## Acknowledgements

This project is inspired by and based on [mohamed-badaoui/asus-touchpad-numpad-driver](https://github.com/mohamed-badaoui/asus-touchpad-numpad-driver). Many thanks to all the contributors of that project for their pioneering work on this functionality.

## License

This project is free software - use, modify and share as you wish.

