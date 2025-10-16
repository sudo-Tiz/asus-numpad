# Asus Touchpad Numpad Driver - Makefile
# Simple and modular build system

.PHONY: all build deps install uninstall clean help

BINARY := asus-numpad
INSTALL_PATH := /usr/local/bin
CONFIG_DIR := /etc/asus-numpad
SERVICE_FILE := asus-numpad.service
SERVICE_PATH := /etc/systemd/system

all: build

build:
	@echo "Building $(BINARY)..."
	@go build -ldflags="-s -w" -o $(BINARY) .
	@echo "✓ Build complete"

deps:
	@echo "Installing dependencies..."
	@if command -v apt >/dev/null 2>&1; then \
		echo "Detected apt (Debian/Ubuntu)"; \
		sudo apt update && sudo apt install -y i2c-tools; \
	elif command -v pacman >/dev/null 2>&1; then \
		echo "Detected pacman (Arch)"; \
		sudo pacman -S --noconfirm i2c-tools; \
	elif command -v dnf >/dev/null 2>&1; then \
		echo "Detected dnf (Fedora)"; \
		sudo dnf install -y i2c-tools; \
	elif command -v zypper >/dev/null 2>&1; then \
		echo "Detected zypper (openSUSE)"; \
		sudo zypper install -y i2c-tools; \
	else \
		echo "⚠ Unknown package manager. Please install i2c-tools manually"; \
		exit 1; \
	fi
	@echo "✓ Dependencies installed"

install: build
	@echo "Installing $(BINARY)..."
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "⚠ Installation requires root privileges"; \
		echo "Please run: sudo make install"; \
		exit 1; \
	fi
	@echo "→ Loading i2c-dev module..."
	@modprobe i2c-dev || (echo "⚠ Failed to load i2c-dev module" && exit 1)
	@echo "→ Installing binary to $(INSTALL_PATH)..."
	@install -m 755 $(BINARY) $(INSTALL_PATH)/
	@echo "→ Installing configuration to $(CONFIG_DIR)..."
	@mkdir -p $(CONFIG_DIR)
	@install -m 644 layout.json $(CONFIG_DIR)/
	@echo "→ Installing systemd service..."
	@install -m 644 $(SERVICE_FILE) $(SERVICE_PATH)/
	@echo "→ Configuring i2c-dev to load at boot..."
	@echo "i2c-dev" > /etc/modules-load.d/i2c-dev.conf
	@echo "→ Reloading systemd..."
	@systemctl daemon-reload
	@echo "→ Enabling service..."
	@systemctl enable $(BINARY)
	@echo "→ Starting service..."
	@systemctl restart $(BINARY)
	@echo "✓ Installation complete!"
	@echo ""
	@echo "Usage: Tap top-right corner to toggle numpad"
	@echo "Logs:  journalctl -fu $(BINARY)"

uninstall:
	@echo "Uninstalling $(BINARY)..."
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "⚠ Uninstallation requires root privileges"; \
		echo "Please run: sudo make uninstall"; \
		exit 1; \
	fi
	@echo "→ Stopping service..."
	@systemctl stop $(BINARY) 2>/dev/null || true
	@echo "→ Disabling service..."
	@systemctl disable $(BINARY) 2>/dev/null || true
	@echo "→ Removing service file..."
	@rm -f $(SERVICE_PATH)/$(SERVICE_FILE)
	@echo "→ Removing binary..."
	@rm -f $(INSTALL_PATH)/$(BINARY)
	@echo "→ Removing configuration..."
	@rm -rf $(CONFIG_DIR)
	@echo "→ Removing i2c-dev module config..."
	@rm -f /etc/modules-load.d/i2c-dev.conf
	@echo "→ Reloading systemd..."
	@systemctl daemon-reload
	@echo "✓ Uninstall complete"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY)
	@echo "✓ Clean complete"

help:
	@echo "Asus Touchpad Numpad Driver - Makefile targets"
	@echo ""
	@echo "  make build      - Build the binary"
	@echo "  make deps       - Install i2c-tools dependency"
	@echo "  make install    - Install binary, config and service (requires sudo)"
	@echo "  make uninstall  - Remove all installed files (requires sudo)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make help       - Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  make deps && sudo make install"
