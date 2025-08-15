#!/bin/bash

# Function to check if script is running as root
check_root() {
    if [[ $(id -u) != 0 ]]; then
        echo "Please run the installation script as root (using sudo for example)"
        exit 1
    fi
}

# Function to install required dependencies based on package manager
install_dependencies() {
    if [[ $(sudo apt install 2>/dev/null) ]]; then
        echo "Installing dependencies with apt..."
        sudo apt -y install libevdev2 python3-libevdev i2c-tools git
    elif [[ $(sudo pacman -h 2>/dev/null) ]]; then
        echo "Installing dependencies with pacman..."
        sudo pacman --noconfirm -S libevdev python-libevdev i2c-tools git
    elif [[ $(sudo dnf install 2>/dev/null) ]]; then
        echo "Installing dependencies with dnf..."
        sudo dnf -y install libevdev python-libevdev i2c-tools git
    else
        echo "Unsupported package manager. Please install these packages manually:"
        echo "- libevdev"
        echo "- python-libevdev/python3-libevdev"
        echo "- i2c-tools"
        echo "- git"
        exit 1
    fi
}

# Function to configure and check i2c
check_i2c() {
    echo "Loading i2c-dev module..."
    modprobe i2c-dev
    
    # Check if module loaded successfully
    if [[ $? != 0 ]]; then
        echo "i2c-dev module cannot be loaded correctly. Make sure you have installed i2c-tools package"
        exit 1
    fi
    
    # Find i2c interfaces
    interfaces=$(for i in $(i2cdetect -l | grep DesignWare | sed -r "s/^(i2c\-[0-9]+).*/\1/"); do echo $i; done)
    if [ -z "$interfaces" ]; then
        echo "No interface i2c found. Make sure you have installed libevdev packages"
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

# Clean up any previous Python cache files
cleanup_cache() {
    if [[ -d numpad_layouts/__pycache__ ]]; then
        rm -rf numpad_layouts/__pycache__
    fi
}

# Function to get available models from numpad_layouts directory
get_available_models() {
    # Get list of Python files and remove .py extension
    local models=()
    for file in numpad_layouts/*.py; do
        # Skip __init__.py or any other special files
        if [[ $(basename "$file") != __* ]]; then
            models+=($(basename "$file" .py))
        fi
    done
    echo "${models[@]}"
}

# Function to select model
select_model() {
    local available_models=($(get_available_models))
    
    echo
    echo "Select models keypad layout:"
    PS3='Please enter your choice: '
    
    # Add Quit option to the list
    options=("${available_models[@]}" "Quit")
    
    select opt in "${options[@]}"; do
        if [[ $opt == "Quit" ]]; then
            echo "Installation cancelled."
            exit 0
        elif [[ " ${available_models[*]} " =~ " ${opt} " ]]; then
            model=$opt
            break
        else
            echo "Invalid option $REPLY. Please try again."
        fi
    done
}

# Function to select keyboard layout
select_keyboard_layout() {
    echo
    echo "What is your keyboard layout?"
    PS3='Please enter your choice: '
    options=("Qwerty" "Azerty" "Quit")
    
    select opt in "${options[@]}"; do
        case $opt in
            "Qwerty")
                percentage_key=6 # Number 5
                break
                ;;
            "Azerty")
                percentage_key=40 # Apostrophe key
                break
                ;;
            "Quit")
                echo "Installation cancelled."
                exit 0
                ;;
            *) 
                echo "Invalid option $REPLY. Please try again."
                ;;
        esac
    done
}

# Function to install service files
install_service() {
    echo "Creating directory structure..."
    mkdir -p /usr/share/asus_touchpad_numpad-driver/numpad_layouts
    mkdir -p /var/log/asus_touchpad_numpad-driver
    
    echo "Installing service files..."
    # Use main.py instead of asus_touchpad.py if it exists
    if [[ -f "main.py" ]]; then
        install main.py /usr/share/asus_touchpad_numpad-driver/asus_touchpad.py
    else
        install asus_touchpad.py /usr/share/asus_touchpad_numpad-driver/
    fi
    
    install -t /usr/share/asus_touchpad_numpad-driver/numpad_layouts numpad_layouts/*.py
    
    echo "Setting up systemd service..."
    cat asus_touchpad.service | LAYOUT=$model PERCENTAGE_KEY=$percentage_key envsubst '$LAYOUT $PERCENTAGE_KEY' > /etc/systemd/system/asus_touchpad_numpad.service
    
    echo "Ensuring i2c-dev is loaded at boot..."
    echo "i2c-dev" | tee /etc/modules-load.d/i2c-dev.conf >/dev/null
}

# Function to enable and start service
start_service() {
    echo "Enabling asus_touchpad_numpad service..."
    systemctl enable asus_touchpad_numpad
    
    if [[ $? != 0 ]]; then
        echo "Something went wrong while enabling asus_touchpad_numpad.service"
        exit 1
    else
        echo "Asus touchpad service enabled"
    fi
    
    echo "Starting asus_touchpad_numpad service..."
    systemctl restart asus_touchpad_numpad
    
    if [[ $? != 0 ]]; then
        echo "Something went wrong while starting asus_touchpad_numpad.service"
        exit 1
    else
        echo "Asus touchpad service started successfully"
    fi
}

# Main installation process
main() {
    check_root
    install_dependencies
    check_i2c
    cleanup_cache
    select_model
    select_keyboard_layout
    install_service
    start_service
    
    echo
    echo "Installation completed successfully!"
    echo "To toggle the numpad, tap the top right corner of your touchpad."
    exit 0
}

# Run the main function
main

