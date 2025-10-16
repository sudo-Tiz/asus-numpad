#!/bin/bash

# Show usage information
show_usage() {
    echo "Usage: $0 [install|uninstall]"
    echo "  install   - Install the Asus touchpad numpad driver"
    echo "  uninstall - Uninstall the driver and clean up all files"
    exit 1
}

# Function to check if script is running as root
check_root() {
    if [[ $(id -u) != 0 ]]; then
        echo "Please run this script as root (using sudo for example)"
        exit 1
    fi
}

# Function to check if a package is installed
check_package() {
    local package=$1
    local manager=$2
    
    if [[ "$manager" == "apt" ]]; then
        dpkg -s "$package" &> /dev/null
        return $?
    elif [[ "$manager" == "pacman" ]]; then
        pacman -Qi "$package" &> /dev/null
        return $?
    elif [[ "$manager" == "dnf" ]]; then
        rpm -q "$package" &> /dev/null
        return $?
    fi
    return 1
}

# Function to handle package installation for a specific package manager
process_packages() {
    local manager=$1
    local install_cmd=$2
    local packages=("${@:3}")
    local missing_packages=()
    
    echo "Checking dependencies for $manager..."
    
    for package in "${packages[@]}"; do
        if ! check_package "$package" "$manager"; then
            missing_packages+=("$package")
        fi
    done
    
    if [[ ${#missing_packages[@]} -eq 0 ]]; then
        echo "All required packages are already installed."
        return 0
    else
        echo "Installing missing packages: ${missing_packages[*]}"
        eval "$install_cmd ${missing_packages[*]}"
        return $?
    fi
}

# Function to install required dependencies based on package manager
install_dependencies() {
    # Only i2c-tools is needed for the Go version
    if [[ $(sudo apt install 2>/dev/null) ]]; then
        process_packages "apt" "sudo apt -y install" "i2c-tools" "golang"
    elif [[ $(sudo pacman -h 2>/dev/null) ]]; then
        process_packages "pacman" "sudo pacman --noconfirm -S" "i2c-tools" "go"
    elif [[ $(sudo dnf install 2>/dev/null) ]]; then
        process_packages "dnf" "sudo dnf -y install" "i2c-tools" "golang"
    else
        echo "Unsupported package manager. Please install these packages manually:"
        echo "- i2c-tools"
        echo "- go/golang"
        exit 1
    fi
}

# Function to configure and check i2c
check_i2c() {
    echo "Loading i2c-dev module..."
    modprobe i2c-dev
    
    if [[ $? != 0 ]]; then
        echo "i2c-dev module cannot be loaded correctly. Make sure you have installed i2c-tools package"
        exit 1
    fi
    
    # Find i2c interfaces
    interfaces=$(for i in $(i2cdetect -l | grep DesignWare | sed -r "s/^(i2c\-[0-9]+).*/\1/"); do echo $i; done)
    if [ -z "$interfaces" ]; then
        echo "No interface i2c found. Make sure you have installed i2c-tools"
        exit 1
    fi
    
    # Test interfaces to find the touchpad
    touchpad_detected=false
    for i in $interfaces; do
        echo -n "Testing interface $i : "
        number=$(echo -n $i | cut -d'-' -f2)
        offTouchpadCmd="i2ctransfer -f -y $number w13@0x15 0x05 0x00 0x3d 0x03 0x06 0x00 0x07 0x00 0x0d 0x14 0x03 0x00 0xad"
        i2c_test=$($offTouchpadCmd 2>&1)
        if [ -z "$i2c_test" ]; then
            echo "success"
            touchpad_detected=true
            break
        else
            echo "failed"
        fi
    done
    
    if [ "$touchpad_detected" = false ]; then
        echo 'The detection was not successful. Touchpad not found.'
        exit 1
    fi
}

# Function to build the Go binary
build_binary() {
    echo "Building Asus touchpad numpad driver..."
    
    cd "$(dirname "$0")"
    
    go build -ldflags="-s -w" -o asus-numpad .
    
    if [[ $? != 0 ]]; then
        echo "Failed to build the Go binary"
        exit 1
    fi
    
    echo "Build completed successfully"
}

# Function to install service files
install_service() {
    echo "Creating directory structure..."
    mkdir -p /usr/local/bin
    mkdir -p /etc/asus-numpad
    
    echo "Installing binary and configuration..."
    install -m 755 asus-numpad /usr/local/bin/
    install -m 644 layout.json /etc/asus-numpad/
    
    echo "Setting up systemd service..."
    cat > /etc/systemd/system/asus-numpad.service << 'EOF'
[Unit]
Description=Asus Touchpad Numpad Driver
After=multi-user.target

[Service]
Type=simple
ExecStart=/usr/local/bin/asus-numpad
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    
    echo "Ensuring i2c-dev is loaded at boot..."
    echo "i2c-dev" | tee /etc/modules-load.d/i2c-dev.conf >/dev/null
    
    systemctl daemon-reload
}

# Function to enable and start service
start_service() {
    echo "Enabling asus-numpad service..."
    systemctl enable asus-numpad
    
    if [[ $? != 0 ]]; then
        echo "Something went wrong while enabling asus-numpad.service"
        exit 1
    fi
    
    echo "Starting asus-numpad service..."
    systemctl restart asus-numpad
    
    if [[ $? != 0 ]]; then
        echo "Something went wrong while starting asus-numpad.service"
        exit 1
    fi
    
    echo "Service started successfully!"
}

# Function to uninstall everything
uninstall() {
    echo "Uninstalling Asus touchpad numpad driver..."
    
    # Stop and disable service
    systemctl stop asus-numpad 2>/dev/null
    systemctl disable asus-numpad 2>/dev/null
    
    # Remove service file
    rm -f /etc/systemd/system/asus-numpad.service
    
    # Remove binary and config
    rm -f /usr/local/bin/asus-numpad
    rm -rf /etc/asus-numpad
    
    # Remove i2c module configuration
    rm -f /etc/modules-load.d/i2c-dev.conf
    
    # Unload i2c module
    modprobe -r i2c-dev 2>/dev/null
    
    # Reload systemd
    systemctl daemon-reload
    
    echo "Uninstall completed successfully!"
}

# Main installation process
install() {
    check_root
    install_dependencies
    check_i2c
    build_binary
    install_service
    start_service
    
    echo
    echo "Installation completed successfully!"
    echo "To toggle the numpad, tap the top right corner of your touchpad."
    echo "To view logs: journalctl -fu asus-numpad"
}

# Main function
main() {
    check_root
    
    case "${1:-install}" in
        install)
            install
            ;;
        uninstall)
            uninstall
            ;;
        *)
            show_usage
            ;;
    esac
}

# Run the main function with all arguments
main "$@"
