#!/bin/bash
# Hubble Network Installer Download and Run Script
# Usage: 
#   With credentials: curl -fsSL https://hubble.com/install.sh | bash -s <base64-credentials>
#   Without credentials: curl -fsSL https://hubble.com/install.sh | bash

set -e

# Accept credentials as first argument (base64 encoded org_id:api_key)
if [ -n "$1" ]; then
    VALIDATION_FAILED=0
    
    # Validate base64 format
    if ! echo "$1" | base64 -d > /dev/null 2>&1; then
        VALIDATION_FAILED=1
    else
        # Decode and validate format (should contain a colon)
        DECODED=$(echo "$1" | base64 -d 2>/dev/null)
        if ! echo "$DECODED" | grep -q ':'; then
            VALIDATION_FAILED=1
        fi
    fi
    
    if [ $VALIDATION_FAILED -eq 1 ]; then
        echo ""
        echo "⚠️  We were unable to validate your credentials."
        echo ""
        echo "You can either:"
        echo "  • Exit and check that you pasted the complete command correctly"
        echo "  • Continue and enter your credentials manually"
        echo ""
        read -p "Would you like to exit and try again? (Y/n): " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
            echo "Please check your command and run the installer again."
            exit 1
        fi
        echo "Continuing - you'll be prompted for credentials..."
        echo ""
    else
        export HUBBLE_CREDENTIALS="$1"
        echo "✓ Credentials provided"
    fi
fi

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

