# Asus Touchpad Numpad Driver# Asus Touchpad Numpad Driver



A simple, lightweight Linux driver written in Go to enable the numpad functionality on Asus laptops with touchpad-integrated numpads. Toggle between normal touchpad usage and numpad mode with a simple touch.A simple, lightweight Linux driver written in Go to enable the numpad functionality on Asus laptops with touchpad-integrated numpads. Toggle between normal touchpad usage and numpad mode with a simple touch.



## TLDR## TLDR



```bash```bash

git clone https://github.com/sudo-Tiz/asus-numpad.git && cd asus-numpad && make installgit clone https://github.com/sudo-Tiz/asus-numpad.git && cd asus-numpad && make install

``````



## Features## Features



- **Simple**: Pure Go implementation, single static binary- **Simple**: Pure Go implementation, single static binary

- **Fast**: Low resource usage, minimal CPU/memory footprint- **Fast**: Low resource usage, no Python runtime required

- **Flexible**: Easy-to-edit JSON layout configuration- **Flexible**: Easy-to-edit JSON layout configuration

- **Lightweight**: Minimal dependencies (only i2c-tools)- **Lightweight**: Minimal dependencies (only i2c-tools)

- Toggle numpad mode by tapping the top-right corner of the touchpad- Toggle numpad mode by tapping the top-right corner of the touchpad



## Configuration## Configuration



The numpad layout is defined in `/etc/asus-numpad/layout.json`. You can customize it to match your specific touchpad layout. The default configuration is for the VivoBook M433IA model.The numpad layout is defined in `/etc/asus-numpad/layout.json`. You can customize it to match your specific touchpad layout. The default configuration is for the VivoBook M433IA model. 



Example layout structure:## Requirements

```json

{- Linux distribution with systemd

  "name": "m433ia",- i2c-tools package

  "cols": 5,- libevdev2 and python3-libevdev packages

  "rows": 4,

  "top_offset": 0.3,## Installation

  "keys": [

    ["KEY_KP7", "KEY_KP8", "KEY_KP9", "KEY_KPSLASH", "KEY_BACKSPACE"],### 1. Install Dependencies

    ["KEY_KP4", "KEY_KP5", "KEY_KP6", "KEY_KPASTERISK", "KEY_BACKSPACE"],

    ...For Debian/Ubuntu-based distributions:

  ]```bash

}sudo apt install libevdev2 python3-libevdev i2c-tools git

``````



## RequirementsFor Arch-based distributions:

```bash

- Linux distribution with systemdsudo pacman -S libevdev python-libevdev i2c-tools git

- i2c-tools package```

- Go 1.16+ (only for building, not needed for running)

For Fedora:

## Installation```bash

sudo dnf install libevdev python-libevdev i2c-tools git

### Quick Install```



```bash### 2. Install the Driver

git clone https://github.com/sudo-Tiz/asus-numpad.git

cd asus-numpadClone this repository and run the installation:

make install```bash

```git clone https://github.com/sudo-Tiz/asus-numpad.git

cd asus-numpad

The installation script will:make install

1. Install dependencies (i2c-tools, Go compiler if needed)```

2. Detect and test your touchpad

3. Build the Go binaryThe installation process will:

4. Install the binary to `/usr/local/bin/asus-numpad`1. Check for dependencies

5. Install the default layout to `/etc/asus-numpad/layout.json`2. Detect and test your touchpad

6. Set up and enable the systemd service3. Ask you to select your laptop model

4. Ask for your keyboard layout (QWERTY or AZERTY)

### Manual Build5. Set up and enable the systemd service



If you want to just build the binary:## Usage

```bash

make build- **Toggle numpad mode**: Tap the top-right corner of your touchpad

# or- **Launch calculator**: Tap the top-left corner of your touchpad

go build -o asus-numpad .

```## Troubleshooting



### Running Manually### Viewing Logs



You can run the driver manually (requires root):To see the service logs:

```bash```bash

sudo ./asus-numpadjournalctl -u asus_touchpad_numpad

# or with custom layout```

sudo ./asus-numpad --layout-file /path/to/layout.json

# or with debug loggingFor real-time log viewing:

sudo ./asus-numpad --debug```bash

```journalctl -fu asus_touchpad_numpad

```

## Usage

### Enable Debug Logging

- **Toggle numpad mode**: Tap the top-right corner of your touchpad

To run the script with debug logging:

When numpad mode is active:```bash

- The touchpad LED will light upLOG=DEBUG sudo -E /usr/share/asus_touchpad_numpad-driver/asus_touchpad.py

- Touch positions on the touchpad are mapped to numpad keys according to your layout```

- Tap the top-right corner again to return to normal touchpad mode

### Boot Failure

## Troubleshooting

If the service fails to start at boot (common on some distributions like Pop!_OS, Linux Mint, Elementary OS, or Solus OS), you can increase the sleep time in the service file:

### Viewing Logs

```bash

To see the service logs:sudo nano /etc/systemd/system/asus_touchpad_numpad.service

```bash```

journalctl -u asus-numpad

```Adjust the ExecStartPre line to increase the delay:

```

For real-time log viewing:ExecStartPre=/bin/sleep 5

```bash```

journalctl -fu asus-numpad

```### Uninstallation



### Enable Debug LoggingTo uninstall:

```bash

Edit the service file to enable debug mode:make uninstall

```bash```

sudo systemctl edit asus-numpad

```## Adding New Layouts



Add:To add support for a new laptop model, create a new Python file in the `numpad_layouts` directory with the appropriate key layout configuration.

```ini

[Service]## Acknowledgements

ExecStart=

ExecStart=/usr/local/bin/asus-numpad --debugThis project is inspired by and based on [mohamed-badaoui/asus-touchpad-numpad-driver](https://github.com/mohamed-badaoui/asus-touchpad-numpad-driver). Many thanks to all the contributors of that project for their pioneering work on this functionality.

```

## License

Then restart:

```bashThis project is free software - use, modify and share as you wish.

sudo systemctl restart asus-numpad

```

### Touchpad Not Detected

If the driver fails to detect your touchpad:
1. Check that i2c-tools is installed: `i2cdetect -l`
2. Verify the i2c-dev module is loaded: `lsmod | grep i2c_dev`
3. Run with debug logging to see detection attempts

### Custom Layout

To customize the layout for your specific model:
1. Edit `/etc/asus-numpad/layout.json`
2. Adjust `cols`, `rows`, and `top_offset` to match your touchpad
3. Modify the `keys` array to map positions to key codes
4. Restart the service: `sudo systemctl restart asus-numpad`

### Uninstallation

```bash
make uninstall
```

## Command-Line Options

```
Usage of asus-numpad:
  -layout-file string
        Path to layout JSON file (default "/etc/asus-numpad/layout.json")
  -debug
        Enable debug logging
```

## Project Structure

```
.
├── main.go          # Entry point and configuration loading
├── driver.go        # Core driver logic and event handling
├── devices.go       # Device detection and parsing
├── layout.json      # Default numpad layout
├── install.sh       # Installation script
├── Makefile         # Build and install automation
└── README.md        # This file
```

## Why Go?

The original Python version was rewritten in Go for several reasons:
- **Performance**: Go is compiled and has much lower runtime overhead
- **Simplicity**: Single static binary, no runtime dependencies
- **Reliability**: Strong typing and better error handling
- **Distribution**: Easy to build and distribute across different systems

## Acknowledgements

This project is inspired by and based on [mohamed-badaoui/asus-touchpad-numpad-driver](https://github.com/mohamed-badaoui/asus-touchpad-numpad-driver). Many thanks to all the contributors of that project for their pioneering work on this functionality.

## License

This project is free software - use, modify and share as you wish.
