#!/bin/bash
# Hubble Network Installer Download and Run Script
# Usage: curl -fsSL https://hubble.com/install.sh | bash

set -e

INSTALL_URL="https://hubble-install.s3.amazonaws.com"
BINARY_NAME="hubble-install"

echo "🛰️  Hubble Network Installer"
echo "=============================="
echo ""

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS="windows"
        BINARY_NAME="hubble-install.exe"
        ;;
    *)
        echo "❌ Error: Unsupported operating system: $OS"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

DOWNLOAD_FILE="hubble-install-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    DOWNLOAD_FILE="hubble-install-${OS}-${ARCH}.exe"
fi

DOWNLOAD_URL="${INSTALL_URL}/${DOWNLOAD_FILE}"

echo "✓ Detected platform: ${OS}/${ARCH}"
echo "📥 Downloading installer..."
echo ""

# Download the binary to temp location
TEMP_FILE=$(mktemp)
if command -v curl > /dev/null 2>&1; then
    if ! curl -fsSL "${DOWNLOAD_URL}" -o "${TEMP_FILE}"; then
        echo "❌ Download failed from S3"
        exit 1
    fi
elif command -v wget > /dev/null 2>&1; then
    if ! wget -q "${DOWNLOAD_URL}" -O "${TEMP_FILE}"; then
        echo "❌ Download failed from S3"
        exit 1
    fi
else
    echo "❌ Error: Neither curl nor wget found. Please install one and try again."
    exit 1
fi

# Make it executable
chmod +x "${TEMP_FILE}"

echo "✓ Download complete!"
echo "🚀 Running installer..."
echo ""

# Run the installer directly from temp location
"${TEMP_FILE}"

# Clean up
rm -f "${TEMP_FILE}"

