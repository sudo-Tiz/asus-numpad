# Makefile for Asus Touchpad Numpad Driver
# Keep It Simple, Stupid (KISS) approach

.PHONY: build install uninstall help

# Default target
all: build

# Build the Go binary
build:
	@echo "Building Asus Touchpad Numpad Driver..."
	@go build -ldflags="-s -w" -o asus-numpad .

# Install the driver
install:
	@echo "Installing Asus Touchpad Numpad Driver..."
	@sudo ./install.sh install

# Uninstall the driver
uninstall:
	@echo "Uninstalling Asus Touchpad Numpad Driver..."
	@sudo ./install.sh uninstall

# Show help
help:
	@echo "Asus Touchpad Numpad Driver"
	@echo "Targets: build install uninstall help"