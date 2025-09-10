#!/bin/bash
set -e

# TimeMachine CLI Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/deepakkumarnarayana/timemachine-cli/main/install.sh | bash

REPO="deepakkumarnarayana/timemachine-cli"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="timemachine"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if running as root for system install
check_sudo() {
    if [ "$EUID" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
        SUDO=""
    elif command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        print_warning "sudo not available, installing to ~/.local/bin"
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        SUDO=""
    fi
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="macos"
            ;;
        mingw*|cygwin*|msys*)
            print_error "Windows is not supported by this install script. Please download the binary manually."
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
}

# Validate version format (e.g., v1.2.3)
validate_version() {
    local version="$1"
    # Allow only alphanumeric, dots, and hyphens in version strings
    if ! echo "$version" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$'; then
        return 1
    fi
    return 0
}

# Get latest release version
get_latest_version() {
    print_status "Fetching latest release information..."
    
    # Use secure TLS and validate response
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s --max-time 30 --fail \
            "https://api.github.com/repos/$REPO/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"tag_name": "([^"]+)".*/\1/' | \
            head -n 1)
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- --timeout=30 --secure-protocol=TLSv1_2 \
            "https://api.github.com/repos/$REPO/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"tag_name": "([^"]+)".*/\1/' | \
            head -n 1)
    else
        print_error "Neither curl nor wget is available. Please install one of them."
    fi
    
    if [ -z "$VERSION" ]; then
        print_error "Failed to get latest release version"
    fi
    
    # Validate version format for security
    if ! validate_version "$VERSION"; then
        print_error "Invalid version format received: $VERSION"
    fi
    
    print_status "Latest version: $VERSION"
}

# Validate URL safety
validate_url() {
    local url="$1"
    # Ensure URL is from GitHub releases only
    if ! echo "$url" | grep -qE '^https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+/releases/download/v[0-9]+\.[0-9]+\.[0-9]+'; then
        return 1
    fi
    return 0
}

# Download and install binary
download_and_install() {
    BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/timemachine-$PLATFORM"
    CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/timemachine-$PLATFORM.sha256"
    
    # Validate URLs for security
    if ! validate_url "$BINARY_URL" || ! validate_url "$CHECKSUM_URL"; then
        print_error "Invalid or unsafe download URLs"
    fi
    
    # Create secure temporary directory
    TEMP_DIR=$(mktemp -d -t timemachine-install.XXXXXX)
    if [ ! -d "$TEMP_DIR" ]; then
        print_error "Failed to create secure temporary directory"
    fi
    
    # Set restrictive permissions
    chmod 700 "$TEMP_DIR"
    
    BINARY_PATH="$TEMP_DIR/timemachine-$PLATFORM"
    CHECKSUM_PATH="$TEMP_DIR/timemachine-$PLATFORM.sha256"
    
    print_status "Downloading TimeMachine CLI $VERSION for $PLATFORM..."
    
    # Download binary with security measures
    if command -v curl >/dev/null 2>&1; then
        # Use secure curl options
        curl -fsSL --max-time 300 --max-filesize 50000000 \
             --proto '=https' --tlsv1.2 \
             "$BINARY_URL" -o "$BINARY_PATH" || print_error "Failed to download binary"
        curl -fsSL --max-time 30 --max-filesize 1000 \
             --proto '=https' --tlsv1.2 \
             "$CHECKSUM_URL" -o "$CHECKSUM_PATH" || print_error "Failed to download checksum"
    elif command -v wget >/dev/null 2>&1; then
        # Use secure wget options  
        wget -q --timeout=300 --secure-protocol=TLSv1_2 --https-only \
             --max-redirect=3 --quota=50M \
             "$BINARY_URL" -O "$BINARY_PATH" || print_error "Failed to download binary"
        wget -q --timeout=30 --secure-protocol=TLSv1_2 --https-only \
             --max-redirect=3 --quota=1K \
             "$CHECKSUM_URL" -O "$CHECKSUM_PATH" || print_error "Failed to download checksum"
    fi
    
    # Verify checksum
    print_status "Verifying checksum..."
    cd "$TEMP_DIR"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "timemachine-$PLATFORM.sha256" >/dev/null || print_error "Checksum verification failed"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 -c "timemachine-$PLATFORM.sha256" >/dev/null || print_error "Checksum verification failed"
    else
        print_warning "No checksum utility found, skipping verification"
    fi
    
    # Install binary
    print_status "Installing to $INSTALL_DIR..."
    chmod +x "$BINARY_PATH"
    $SUDO cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME" || print_error "Failed to install binary"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    print_success "TimeMachine CLI installed successfully!"
}

# Check if already installed
check_existing() {
    if command -v timemachine >/dev/null 2>&1; then
        EXISTING_VERSION=$(timemachine --version 2>/dev/null | head -n1 || echo "unknown")
        print_warning "TimeMachine CLI is already installed: $EXISTING_VERSION"
        read -p "Do you want to continue and update? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled"
            exit 0
        fi
    fi
}

# Main installation process
main() {
    echo
    echo "🕰️  TimeMachine CLI Installer"
    echo "================================"
    echo
    
    check_existing
    detect_platform
    print_status "Detected platform: $PLATFORM"
    
    check_sudo
    print_status "Install directory: $INSTALL_DIR"
    
    get_latest_version
    download_and_install
    
    echo
    print_success "Installation complete!"
    echo
    echo "🚀 Quick Start:"
    echo "  timemachine init     # Initialize in your Git repository"
    echo "  timemachine start    # Start watching for changes"
    echo "  timemachine --help   # Show all commands"
    echo
    echo "📚 Documentation: https://github.com/$REPO"
    
    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]] && [ "$INSTALL_DIR" != "/usr/local/bin" ]; then
        echo
        print_warning "⚠️  $INSTALL_DIR is not in your PATH"
        echo "Add this to your ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    echo
}

# Run main function
main "$@"