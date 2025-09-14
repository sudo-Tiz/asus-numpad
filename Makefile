# Makefile for Asus Touchpad Numpad Driver
# Keep It Simple, Stupid (KISS) approach

.PHONY: install uninstall help

# Default target
all: install

# Install the driver
install:
	@echo "Installing Asus Touchpad Numpad Driver..."
	@sudo ./asus_touchpad.sh install

# Uninstall the driver
uninstall:
	@echo "Uninstalling Asus Touchpad Numpad Driver..."
	@sudo ./asus_touchpad.sh uninstall

# Show help
help:
	@echo "Asus Touchpad Numpad Driver"
	@echo "Targets: install uninstall help"