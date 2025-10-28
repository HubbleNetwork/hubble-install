# Testing Guide for Hubble Installer

## Quick Test (macOS)

### Option 1: Run directly from source
```bash
cd /Users/ryan/repos/hubble-install
go run main.go
```

### Option 2: Run the compiled binary
```bash
cd /Users/ryan/repos/hubble-install
./hubble-install
# or
./bin/hubble-install-darwin-arm64  # for Apple Silicon
./bin/hubble-install-darwin-amd64  # for Intel
```

## What to Expect

The installer will:

1. **Display banner**
   - Shows Hubble Network branding

2. **Detect platform**
   - Should show "macOS"

3. **Prompt for credentials**
   - Org ID (visible input)
   - API Token (hidden input)

4. **Board selection**
   - Shows 5 available boards:
     1. nRF21540 DK
     2. nRF52840 DK
     3. nRF52 DK
     4. xG22 EK4108A
     5. xG24 EK2703A

5. **Check prerequisites**
   - Checks for: Homebrew, uv, segger-jlink
   - Prompts to install if missing

6. **Install dependencies** (if needed)
   - Installs Homebrew (if missing)
   - Installs uv and segger-jlink in parallel

7. **Flash the board**
   - Runs: `uvx --from pyhubbledemo hubbledemo flash <board> -o <org_id> -t <api_token>`

8. **Shows completion**
   - Total time taken
   - Next steps

## Test Scenarios

### Scenario 1: Fresh Install (No Dependencies)
- System with no Homebrew, uv, or segger-jlink
- **Expected**: Should install all dependencies and flash board
- **Time**: ~2-5 minutes (first time Homebrew install)

### Scenario 2: Partial Install (Homebrew exists)
- System with Homebrew but missing uv/segger-jlink
- **Expected**: Should install only missing dependencies
- **Time**: ~30-60 seconds

### Scenario 3: All Dependencies Present
- System with all dependencies installed
- **Expected**: Should skip dependency installation, go straight to flashing
- **Time**: ~10-15 seconds

### Scenario 4: Interrupt Testing
- Press Ctrl+C during execution
- **Expected**: Clean exit with yellow "cancelled" message

## Dry Run Testing

To test without actually installing/flashing, you can:

1. Cancel when prompted for dependencies (press 'n')
2. Use a test org ID and token (it will fail at flash step but you can see the flow)

## Verifying Success

After completion, verify:

```bash
# Check uv is installed
which uv

# Check segger-jlink is installed
which JLinkExe

# Check pyhubbledemo is accessible
uvx --from pyhubbledemo hubbledemo --help
```

## Common Issues

### Issue: "Permission denied" when installing Homebrew
**Solution**: The script will prompt for your password. This is expected.

### Issue: "Command not found" after installation
**Solution**: You may need to restart your terminal or run:
```bash
export PATH="/opt/homebrew/bin:$PATH"  # for Apple Silicon
# or
export PATH="/usr/local/bin:$PATH"     # for Intel
```

### Issue: Board flashing fails
**Possible causes:**
- Invalid Org ID or API Token
- Board not connected
- Wrong board selected
- Permissions issue with USB device

## Performance Testing

Target: < 30 seconds total time (with all dependencies pre-installed)

Breakdown:
- Platform detection: < 1s
- Credential input: ~10s (user input time)
- Board selection: ~5s (user input time)
- Prerequisites check: < 1s
- Board flashing: ~10-15s
- **Total**: ~26-32s

## Cross-Platform Testing

### macOS (Current)
✅ Fully implemented and tested

### Linux (Future)
- Will need to test on Ubuntu, Debian, Fedora, Arch
- Package managers: apt, yum, pacman
- uv installation method may differ
- SEGGER J-Link installation differs

### Windows (Future)
- Will need chocolatey or scoop for package management
- Windows Terminal vs CMD vs PowerShell
- Different path separators
- .exe handling

## Continuous Integration

For CI/CD, you can run:

```bash
# Build all platforms
./scripts/build.sh

# Verify binaries exist
ls -lh bin/

# Test compilation
go test ./...  # (once tests are written)
```

## Next Steps

1. Add unit tests for each package
2. Add integration tests
3. Add mock mode for CI testing
4. Add version flag: `hubble-install --version`
5. Add dry-run mode: `hubble-install --dry-run`

