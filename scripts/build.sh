#!/bin/bash
set -e

# Build script for Hubble installer
# Builds binaries for all supported platforms

VERSION=${VERSION:-"0.1.0"}
BUILD_DIR="bin"

echo "Building Hubble Installer v${VERSION}"
echo "=========================================="

# Create build directory
mkdir -p ${BUILD_DIR}

# Build for macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o ${BUILD_DIR}/hubble-install-darwin-amd64 .

# Build for macOS (Apple Silicon)
echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=${VERSION}" -o ${BUILD_DIR}/hubble-install-darwin-arm64 .

# Build for Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o ${BUILD_DIR}/hubble-install-linux-amd64 .

# Build for Linux (arm64)
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=${VERSION}" -o ${BUILD_DIR}/hubble-install-linux-arm64 .

# Build for Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o ${BUILD_DIR}/hubble-install-windows-amd64.exe .

echo ""
echo "Build complete! Binaries in ${BUILD_DIR}/"
ls -lh ${BUILD_DIR}/

