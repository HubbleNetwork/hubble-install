#!/bin/bash
# Hubble Network Installer Download and Run Script
# Usage: 
#   With credentials: curl -fsSL https://get.hubble.com | bash -s <base64-credentials>
#   Without credentials: curl -fsSL https://get.hubble.com | bash
#   Opt-out of analytics: curl -fsSL https://get.hubble.com | bash -s <base64-credentials> a=false

set -e

# Collect extra arguments to pass to the installer (e.g., a=false for analytics opt-out)
EXTRA_ARGS=""

# Process all arguments
for arg in "$@"; do
    # Check for analytics opt-out flag
    if [ "$arg" = "a=false" ] || [ "$arg" = "--no-analytics" ]; then
        EXTRA_ARGS="$EXTRA_ARGS $arg"
        continue
    fi
    
    # Skip if we already have credentials set
    if [ -n "$HUBBLE_CREDENTIALS" ]; then
        EXTRA_ARGS="$EXTRA_ARGS $arg"
        continue
    fi
    
    # Try to validate as base64 credentials
    VALIDATION_FAILED=0
    
    # Validate base64 format
    if ! echo "$arg" | base64 -d > /dev/null 2>&1; then
        VALIDATION_FAILED=1
    else
        # Decode and validate format (should contain a colon)
        DECODED=$(echo "$arg" | base64 -d 2>/dev/null)
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
        read -p "Would you like to exit and try again? (Y/n): " -n 1 -r < /dev/tty
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
            echo "Please check your command and run the installer again."
            exit 1
        fi
        echo "Continuing - you'll be prompted for credentials..."
        echo ""
    else
        export HUBBLE_CREDENTIALS="$arg"
        echo "✓ Credentials provided"
    fi
done

GITHUB_REPO="HubbleNetwork/hubble-install"
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

echo "✓ Detected platform: ${OS}/${ARCH}"
echo ""

# Determine download URLs
# Try to get latest release version from GitHub API
if command -v curl > /dev/null 2>&1; then
    LATEST_RELEASE=$(curl -sL https://api.github.com/repos/${GITHUB_REPO}/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
else
    LATEST_RELEASE=""
fi

if [ -z "$LATEST_RELEASE" ]; then
    # Fallback to latest download URL (no specific version)
    BINARY_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${DOWNLOAD_FILE}"
    CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/checksums.txt"
    echo "📥 Downloading latest installer..."
else
    # Use specific version
    BINARY_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_RELEASE}/${DOWNLOAD_FILE}"
    CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_RELEASE}/checksums.txt"
    echo "📥 Downloading installer ${LATEST_RELEASE}..."
fi

echo ""

# Create temp directory for downloads
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

# Save the original working directory
ORIGINAL_DIR=$(pwd)

TEMP_BINARY="${TEMP_DIR}/${DOWNLOAD_FILE}"
TEMP_CHECKSUMS="${TEMP_DIR}/checksums.txt"

# Download the binary
if command -v curl > /dev/null 2>&1; then
    if ! curl -fsSL "${BINARY_URL}" -o "${TEMP_BINARY}"; then
        echo "❌ Download failed from GitHub Releases"
        echo "   URL: ${BINARY_URL}"
        exit 1
    fi
elif command -v wget > /dev/null 2>&1; then
    if ! wget -q "${BINARY_URL}" -O "${TEMP_BINARY}"; then
        echo "❌ Download failed from GitHub Releases"
        echo "   URL: ${BINARY_URL}"
        exit 1
    fi
else
    echo "❌ Error: Neither curl nor wget found. Please install one and try again."
    exit 1
fi

echo "✓ Binary downloaded"

# Download checksums
if command -v curl > /dev/null 2>&1; then
    if ! curl -fsSL "${CHECKSUM_URL}" -o "${TEMP_CHECKSUMS}"; then
        echo "❌ Failed to download checksums"
        exit 1
    fi
elif command -v wget > /dev/null 2>&1; then
    if ! wget -q "${CHECKSUM_URL}" -O "${TEMP_CHECKSUMS}"; then
        echo "❌ Failed to download checksums"
        exit 1
    fi
fi

echo "✓ Checksums downloaded"

# Verify checksum
echo "🔒 Verifying checksum..."

# Change to temp directory for checksum verification
cd "${TEMP_DIR}"

# Use shasum (macOS/BSD) or sha256sum (Linux)
if command -v shasum > /dev/null 2>&1; then
    if ! shasum -a 256 -c checksums.txt --ignore-missing --quiet 2>/dev/null; then
        echo "❌ Checksum verification failed!"
        echo "   This could indicate a corrupted download or security issue."
        exit 1
    fi
elif command -v sha256sum > /dev/null 2>&1; then
    if ! sha256sum -c checksums.txt --ignore-missing --quiet 2>/dev/null; then
        echo "❌ Checksum verification failed!"
        echo "   This could indicate a corrupted download or security issue."
        exit 1
    fi
else
    echo "⚠️  Warning: Neither shasum nor sha256sum found. Skipping checksum verification."
    echo "   Install shasum or sha256sum for secure downloads."
fi

echo "✓ Checksum verified"
echo ""

# Make it executable
chmod +x "${TEMP_BINARY}"

# Change back to the original directory before running the installer
cd "${ORIGINAL_DIR}"

echo "🚀 Running installer..."
echo ""

# Run the installer from the user's original working directory
# Pass any extra arguments (e.g., a=false for analytics opt-out)
"${TEMP_BINARY}" ${EXTRA_ARGS}

# Temp directory and files will be cleaned up by trap on exit
