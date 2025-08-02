#!/bin/bash

# Golte Service Installation Script
# This script installs and configures the Golte systemd service

set -e

# Configuration
SERVICE_NAME="golte"
SERVICE_USER="golte"
INSTALL_DIR="/opt/golte"
BINARY_NAME="golte"
CONFIG_FILE="config.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Create service user
create_user() {
    log_info "Creating service user: $SERVICE_USER"
    if id "$SERVICE_USER" &>/dev/null; then
        log_warning "User $SERVICE_USER already exists"
    else
        useradd --system --no-create-home --shell /bin/false --group dialout "$SERVICE_USER"
        log_success "Created user: $SERVICE_USER"
    fi
}

# Create installation directory
create_directories() {
    log_info "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    chown "$SERVICE_USER:dialout" "$INSTALL_DIR"
    chmod 755 "$INSTALL_DIR"
    log_success "Created directory: $INSTALL_DIR"
}

# Build the binary
build_binary() {
    log_info "Building Golte binary"
    if [[ ! -f "go.mod" ]]; then
        log_error "go.mod not found. Please run this script from the project root directory."
        exit 1
    fi
    
    go build -o "$BINARY_NAME" -ldflags "-s -w" .
    if [[ $? -eq 0 ]]; then
        log_success "Binary built successfully"
    else
        log_error "Failed to build binary"
        exit 1
    fi
}

# Install binary and config
install_files() {
    log_info "Installing binary and configuration files"
    
    # Install binary
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    chown "$SERVICE_USER:dialout" "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    
    # Install config if it doesn't exist
    if [[ ! -f "$INSTALL_DIR/$CONFIG_FILE" ]]; then
        if [[ -f "$CONFIG_FILE" ]]; then
            cp "$CONFIG_FILE" "$INSTALL_DIR/"
        elif [[ -f "config.yaml.example" ]]; then
            cp "config.yaml.example" "$INSTALL_DIR/$CONFIG_FILE"
            log_warning "Copied example config. Please edit $INSTALL_DIR/$CONFIG_FILE with your settings"
        else
            log_error "No configuration file found. Please create $INSTALL_DIR/$CONFIG_FILE"
            exit 1
        fi
        chown "$SERVICE_USER:dialout" "$INSTALL_DIR/$CONFIG_FILE"
        chmod 644 "$INSTALL_DIR/$CONFIG_FILE"
    else
        log_info "Configuration file already exists at $INSTALL_DIR/$CONFIG_FILE"
    fi
    
    log_success "Files installed successfully"
}

# Install systemd service
install_service() {
    log_info "Installing systemd service"
    
    if [[ ! -f "golte.service" ]]; then
        log_error "golte.service file not found"
        exit 1
    fi
    
    cp "golte.service" "/etc/systemd/system/"
    systemctl daemon-reload
    log_success "Service installed successfully"
}

# Configure permissions for serial device access
configure_permissions() {
    log_info "Configuring serial device permissions"
    
    # Add user to dialout group (already done during user creation)
    # Create udev rule for consistent device permissions
    cat > /etc/udev/rules.d/99-golte-modem.rules << EOF
# Golte modem device permissions
SUBSYSTEM=="tty", ATTRS{idVendor}=="*", ATTRS{idProduct}=="*", GROUP="dialout", MODE="0664"
KERNEL=="ttyUSB*", GROUP="dialout", MODE="0664"
KERNEL=="ttyACM*", GROUP="dialout", MODE="0664"
KERNEL=="serial*", GROUP="dialout", MODE="0664"
EOF
    
    udevadm control --reload-rules
    udevadm trigger
    log_success "Serial device permissions configured"
}

# Main installation function
main() {
    log_info "Starting Golte service installation"
    
    check_root
    create_user
    create_directories
    build_binary
    install_files
    install_service
    configure_permissions
    
    log_success "Installation completed successfully!"
    echo
    log_info "Next steps:"
    echo "  1. Edit the configuration file: $INSTALL_DIR/$CONFIG_FILE"
    echo "  2. Enable the service: sudo systemctl enable $SERVICE_NAME"
    echo "  3. Start the service: sudo systemctl start $SERVICE_NAME"
    echo "  4. Check service status: sudo systemctl status $SERVICE_NAME"
    echo "  5. View logs: sudo journalctl -u $SERVICE_NAME -f"
}

# Run main function
main "$@"
