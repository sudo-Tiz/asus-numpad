# Asus Touchpad Numpad Driver

[![CI/CD](https://github.com/sudo-Tiz/asus-numpad/actions/workflows/ci.yml/badge.svg)](https://github.com/sudo-Tiz/asus-numpad/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/sudo-Tiz/asus-numpad)](https://goreportcard.com/report/github.com/sudo-Tiz/asus-numpad)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sudo-Tiz/asus-numpad)](go.mod)
[![Release](https://img.shields.io/github/v/release/sudo-Tiz/asus-numpad)](https://github.com/sudo-Tiz/asus-numpad/releases)
[![License](https://img.shields.io/badge/license-Free%20Software-blue.svg)](#license)

A lightweight Linux driver written in Go that enables numpad functionality on Asus laptops with touchpad-integrated numpads.

## Quick Start

```bash
git clone https://github.com/sudo-Tiz/asus-numpad.git
cd asus-numpad
make deps && sudo make install
```

## Features

- **Simple**: Pure Go, single static binary
- **Fast**: Low resource usage, optimized hot path
- **Flexible**: JSON layout configuration
- **Lightweight**: Only dependency is i2c-tools

## Requirements

- i2c-tools package
- Linux with systemd
- Go 1.23+ (for building only)

## Installation

### 1. Install dependencies

The Makefile will detect your package manager automatically:

```bash
make deps
```

Supported package managers: apt, pacman, dnf, zypper

### 2. Build and install

```bash
sudo make install
```

This will:
- Build the binary with optimizations (`-ldflags="-s -w"`)
- Install to `/usr/local/bin/asus-numpad`
- Install layout to `/etc/asus-numpad/layout.json`
- Install and enable systemd service
- Configure i2c-dev module to load at boot

## Usage

**Toggle numpad**: Tap the top-right corner of your touchpad

The touchpad LED will light up when numpad mode is active.

## Configuration

Edit `/etc/asus-numpad/layout.json`:

```json
{
  "try_times": 5,
  "try_sleep_ms": 100,
  "cols": 5,
  "rows": 4,
  "top_offset": 0.3,
  "simulate_numlock": false,
  "keys": [
    ["KEY_KP7", "KEY_KP8", "KEY_KP9", "KEY_KPSLASH", "KEY_BACKSPACE"],
    ["KEY_KP4", "KEY_KP5", "KEY_KP6", "KEY_KPASTERISK", "KEY_BACKSPACE"],
    ["KEY_KP1", "KEY_KP2", "KEY_KP3", "KEY_KPMINUS", "KEY_RESERVED"],
    ["KEY_KP0", "KEY_KPDOT", "KEY_KPENTER", "KEY_KPPLUS", "KEY_KPEQUAL"]
  ]
}
```

After editing, restart the service:

```bash
sudo systemctl restart asus-numpad
```

## Makefile Targets

```bash
make build      # Build the binary
make deps       # Install i2c-tools
make install    # Install everything (requires sudo)
make uninstall  # Remove all files (requires sudo)
make clean      # Remove build artifacts
make help       # Show all targets
```

## Troubleshooting

### Numlock seems to be disabled

When typing on the numpad, if it seems that numlock is disabled, you need to activate it, either with the *Numlock* key or by editing `/etc/asus-numpad/layout.json` and setting `"numlock_is_enabled": false`. In this case, restart the service (`sudo systemctl restart asus-numpad`).

### View logs

```bash
journalctl -fu asus-numpad
```

### Manual run with custom layout

```bash
sudo /usr/local/bin/asus-numpad --layout-file /path/to/layout.json
```

### Uninstall

```bash
sudo make uninstall
```

## Project Structure

```
.
├── main.go              # Entry point and layout loading
├── driver.go            # Core event loop and key mapping
├── devices.go           # Device detection from /proc
├── layout.json          # Default numpad layout (m433ia)
├── asus-numpad.service  # Systemd service file
├── Makefile             # Build automation
└── README.md            # This file
```

## Contributing

### Setup pre-commit hooks

```bash
pip install pre-commit
pre-commit install
```

This will run `gofmt` and `golangci-lint` automatically before each commit.

### Manual lint and format

```bash
pre-commit run --all-files
```

## Acknowledgements

Inspired by [mohamed-badaoui/asus-touchpad-numpad-driver](https://github.com/mohamed-badaoui/asus-touchpad-numpad-driver).

## License

Free software - use, modify and share as you wish.
