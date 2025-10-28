# Hubble Network Smart Installer

Cross-platform installer for Hubble Network developer boards. Flash Nordic and Silicon Labs boards in under 30 seconds.

> **Note:** This installer depends on [pyhubbledemo](https://github.com/HubbleNetwork/pyhubbledemo) for board flashing and requires firmware binaries from [hubble-tldm](https://github.com/HubbleNetwork/hubble-tldm).

## TODO

- [ ] **Add GPG signing for binaries** to verify origin (note: GPG verification code itself can be removed in supply chain attack, so binaries should be self-verified via code signing)
- [ ] **More secure API token handling** - currently passed as CLI argument (visible in process list); consider file descriptor or stdin approach
- [ ] **Linux implementation** - use apt/yum for package management, install uv via pip/curl, and segger-jlink from official packages
- [ ] **Windows implementation** - use Chocolatey or Scoop for package management, install uv and segger-jlink, handle Windows-specific paths and permissions

## Quick Start

### One-Line Install (macOS/Linux)

```bash
curl -fsSL https://hubble.com/install.sh | bash
```

This will:
1. Detect your OS and architecture automatically
2. Download the appropriate binary
3. Run the installer immediately
4. Clean up after completion

### Environment Variables

Set these to skip credential prompts:

```bash
export HUBBLE_ORG_ID="your-org-id"
export HUBBLE_API_TOKEN="your-api-token"
hubble-install
```

Or use them inline:
```bash
HUBBLE_ORG_ID="your-org-id" HUBBLE_API_TOKEN="your-token" hubble-install
```

### Command Line Flags

```bash
hubble-install [flags]

Flags:
  -clean    Remove existing uv and segger-jlink dependencies and clear Homebrew cache, then exit
            (includes verbose debug output by default)
  -debug    Enable debug mode (reserved for future use)
```

**Examples:**

```bash
# Normal installation
hubble-install

# With environment variables (skips prompts)
HUBBLE_ORG_ID="org123" HUBBLE_API_TOKEN="token456" hubble-install

# Clean only (removes dependencies with verbose output and exits)
hubble-install -clean
```

### Manual Installation

#### Download Pre-built Binary

1. Download the appropriate binary for your platform from [Releases](https://github.com/HubbleNetwork/hubble-install/releases)
2. Make it executable: `chmod +x hubble-install-*`
3. Run it: `./hubble-install-*`

#### Build from Source

**Prerequisites:**
- Go 1.21 or later

**Build:**

```bash
# Clone the repository
git clone https://github.com/HubbleNetwork/hubble-install.git
cd hubble-install

# Install dependencies
go mod download

# Build for your platform
go build -o hubble-install .

# Or build for all platforms
chmod +x scripts/build.sh
./scripts/build.sh
```

## Supported Platforms

- ✅ **macOS** (Intel & Apple Silicon) - Full support
- 🚧 **Linux** - Coming soon
- 🚧 **Windows** - Coming soon

## Supported Developer Boards

- **Nordic Semiconductor**
  - nRF21540 DK
  - nRF52840 DK
  - nRF52 DK

- **Silicon Labs**
  - xG22 EK4108A Explorer Kit
  - xG24 EK2703A Explorer Kit

## What It Does

The installer will:

1. 🔍 Detect your operating system
2. 🔑 Prompt for your Hubble credentials (Org ID & API Token)
3. 🎯 Let you select your developer board
4. 📦 Install required dependencies:
   - **macOS**: Homebrew, uv, segger-jlink
   - **Linux**: TBD
   - **Windows**: TBD
5. ⚡ Flash your board using `pyhubbledemo`
6. ✅ Verify the installation

**Total time: < 30 seconds** (after dependencies are installed)

## Getting Your Credentials

Get your Hubble Org ID and API Token from:
👉 https://dash.hubble.com/developer/api-tokens

## Development

### Project Structure

```
hubble-install/
├── main.go                    # Entry point
├── go.mod                     # Go module definition
├── internal/
│   ├── ui/                   # Terminal UI components
│   ├── config/               # Configuration management
│   ├── platform/             # Platform-specific installers
│   └── boards/               # Board definitions
└── scripts/
    ├── build.sh              # Build script
    └── install.sh            # Download script
```

### Running Locally

```bash
# Run directly
go run main.go

# Build and run
go build -o hubble-install .
./hubble-install
```

### Adding a New Board

Edit `internal/boards/boards.go` and add your board to the `AvailableBoards` slice:

```go
{
    ID:          "board_id",
    Name:        "Board Name",
    Description: "Board Description",
    Vendor:      "Vendor Name",
}
```

### Platform Support

To add Linux or Windows support, implement the methods in:
- `internal/platform/linux.go`
- `internal/platform/windows.go`

## Dependencies

### Runtime Dependencies (Installed Automatically)

- **uv** - Fast Python package installer
- **segger-jlink** - SEGGER J-Link tools for board flashing

### Build Dependencies

- Go 1.21+
- `github.com/fatih/color` - Terminal colors
- `golang.org/x/term` - Terminal password input

## License

MIT License - see LICENSE file for details

## Support

- 📚 Documentation: https://docs.hubble.network
- 💬 GitHub Issues: https://github.com/HubbleNetwork/hubble-install/issues
- 🐛 Bug Reports: https://github.com/HubbleNetwork/pyhubbledemo/issues

---

Made with 🛰️ by Hubble Network

