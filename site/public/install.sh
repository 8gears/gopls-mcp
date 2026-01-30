#!/bin/bash
# gopls-mcp installer for Linux and macOS
# Usage: curl -sSL https://raw.githubusercontent.com/[username]/gopls-mcp/main/scripts/install.sh | bash

set -e

REPO="xieyuschen/gopls-mcp"
NAME="gopls-mcp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_os_arch() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)     OS='linux';;
        Darwin*)    OS='darwin';;
        *)          error "Unsupported OS: $OS";;
    esac

    case "$ARCH" in
        x86_64*)    ARCH='amd64';;
        aarch64*)   ARCH='arm64';;
        arm64*)     ARCH='arm64';;
        *)          error "Unsupported architecture: $ARCH";;
    esac

    info "Detected OS: $OS, Architecture: $ARCH"
}

# Get latest release version
get_latest_version() {
    info "Fetching latest release version..."
    VERSION=$(curl -s https://api.github.com/repos/"$REPO"/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version"
    fi

    info "Latest version: $VERSION"
}

# Determine install directory ($HOME/.local/bin)
get_install_dir() {
    INSTALL_DIR="$HOME/.local/bin"

    # Create directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    info "Install directory: $INSTALL_DIR"
}

# Download and install
download_and_install() {
    FILENAME="${NAME}_${VERSION}_${OS}_${ARCH}"
    if [ "$OS" = "linux" ]; then
        URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}.tar.gz"
        TEMP_FILE=$(mktemp).tar.gz
    else
        URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}.tar.gz"
        TEMP_FILE=$(mktemp).tar.gz
    fi

    info "Downloading from: $URL"

    if ! curl -fsSL "$URL" -o "$TEMP_FILE"; then
        error "Failed to download binary"
    fi

    info "Extracting and installing..."
    tar -xzf "$TEMP_FILE" -C "$INSTALL_DIR" "$NAME"
    chmod +x "$INSTALL_DIR/$NAME"

    rm -f "$TEMP_FILE"
}

# Verify installation
verify() {
    if command -v "$NAME" &> /dev/null; then
        VERSION_OUTPUT=$("$NAME" --version 2>&1 || true)
        info "Successfully installed $NAME!"
        info "$VERSION_OUTPUT"
        info "Installation location: $INSTALL_DIR/$NAME"
    else
        warn "Installation completed, but $NAME is not in PATH"
        warn "Add $HOME/.local/bin to your PATH:"
        warn "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc  # or ~/.zshrc"
        warn "  source ~/.bashrc  # or ~/.zshrc"
    fi
}

# Main execution
main() {
    echo ""
    echo "gopls-mcp Installer"
    echo "==================="
    echo ""

    detect_os_arch
    get_latest_version
    get_install_dir
    download_and_install
    verify

    echo ""
    info "Installation complete!"
}

main
