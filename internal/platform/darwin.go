package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// DarwinInstaller implements the Installer interface for macOS
type DarwinInstaller struct{}

// NewDarwinInstaller creates a new macOS installer
func NewDarwinInstaller() *DarwinInstaller {
	return &DarwinInstaller{}
}

// Name returns the platform name
func (d *DarwinInstaller) Name() string {
	return "macOS"
}

// ensureSudoAccess validates sudo access upfront to avoid multiple password prompts
func (d *DarwinInstaller) ensureSudoAccess() error {
	// Check if we already have valid sudo credentials
	checkCmd := exec.Command("sudo", "-n", "true")
	if err := checkCmd.Run(); err == nil {
		// Already have valid sudo, no need to prompt
		return nil
	}

	// Need to prompt for password
	ui.PrintWarning("Administrator access required for installation")
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to obtain sudo access: %w", err)
	}

	return nil
}

// CheckPrerequisites checks for missing dependencies
func (d *DarwinInstaller) CheckPrerequisites() ([]MissingDependency, error) {
	var missing []MissingDependency

	// Check for Homebrew
	if !d.commandExists("brew") {
		missing = append(missing, MissingDependency{
			Name:   "Homebrew",
			Status: "Not installed",
		})
	}

	// Check for uv
	if !d.commandExists("uv") {
		missing = append(missing, MissingDependency{
			Name:   "uv",
			Status: "Not installed",
		})
	}

	// Check for JLink (from segger-jlink)
	if !d.commandExists("JLinkExe") {
		missing = append(missing, MissingDependency{
			Name:   "segger-jlink",
			Status: "Not installed",
		})
	}

	return missing, nil
}

// InstallPackageManager installs Homebrew if not present
func (d *DarwinInstaller) InstallPackageManager() error {
	if d.commandExists("brew") {
		ui.PrintSuccess("Homebrew already installed")
		return nil
	}

	// Ensure we have sudo access upfront (single password prompt)
	// The Homebrew script will use sudo internally when needed (e.g., for Xcode Command Line Tools)
	if err := d.ensureSudoAccess(); err != nil {
		return err
	}

	ui.PrintInfo("Installing Homebrew...")
	ui.PrintInfo("This may take a few minutes...")

	// Run the official Homebrew installation script as regular user (not sudo)
	// The script will internally use sudo when needed, using our cached credentials
	// NONINTERACTIVE=1 suppresses the "running in noninteractive mode" warning
	cmd := exec.Command("/bin/bash", "-c", `NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
	}

	// Add Homebrew to PATH for this process
	if err := d.setupBrewPath(); err != nil {
		return fmt.Errorf("homebrew installation completed but could not find brew binary: %w", err)
	}

	// Verify that brew is actually working
	if !d.commandExists("brew") {
		return fmt.Errorf("homebrew installation completed but brew command not found in PATH")
	}

	// Test brew with a simple command to ensure it's functional
	testCmd := exec.Command("brew", "--version")
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("homebrew installed but not functioning correctly: %w", err)
	}

	ui.PrintSuccess("Homebrew installed successfully")
	return nil
}

// CleanDependencies removes uv and segger-jlink and clears Homebrew cache
func (d *DarwinInstaller) CleanDependencies() error {
	var errors []string

	// Uninstall uv if present
	if d.commandExists("uv") {
		ui.PrintInfo("Removing uv...")

		// Try brew uninstall first
		cmd := exec.Command("brew", "uninstall", "uv", "--force", "--ignore-dependencies")
		if IsDebugMode() {
			ui.PrintDebug("Attempting: brew uninstall uv --force --ignore-dependencies")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Run(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to remove uv: %v", err))
		} else {
			ui.PrintSuccess("uv removed")
		}

		// Remove uv cache
		uvCache := os.ExpandEnv("$HOME/.cache/uv")
		if _, err := os.Stat(uvCache); err == nil {
			if IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Removing cache: %s", uvCache))
			}
			os.RemoveAll(uvCache)
		}
	}

	// Uninstall segger-jlink if present
	if d.commandExists("JLinkExe") {
		ui.PrintInfo("Removing segger-jlink...")
		cmd := exec.Command("brew", "uninstall", "segger-jlink", "--force")
		if IsDebugMode() {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to remove segger-jlink: %v", err))
		} else {
			ui.PrintSuccess("segger-jlink removed")
		}
	}

	// Clear Homebrew cache
	cacheDir := os.ExpandEnv("$HOME/Library/Caches/Homebrew/downloads")
	if _, err := os.Stat(cacheDir); err == nil {
		ui.PrintInfo("Clearing Homebrew cache...")
		if IsDebugMode() {
			ui.PrintDebug(fmt.Sprintf("Removing: %s", cacheDir))
		}
		if err := os.RemoveAll(cacheDir); err != nil {
			errors = append(errors, fmt.Sprintf("failed to clear cache: %v", err))
		} else {
			ui.PrintSuccess("Homebrew cache cleared")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %v", errors)
	}

	return nil
}

// InstallDependencies installs uv and segger-jlink
func (d *DarwinInstaller) InstallDependencies() error {
	// First ensure Homebrew is installed
	if !d.commandExists("brew") {
		if err := d.InstallPackageManager(); err != nil {
			return err
		}
	}

	// Install uv and segger-jlink in parallel for speed
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Install uv
	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.commandExists("uv") {
			ui.PrintSuccess("uv already installed")
			return
		}

		ui.PrintInfo("Installing uv...")
		if err := d.runBrewInstall("uv", false); err != nil {
			errChan <- fmt.Errorf("failed to install uv: %w", err)
			return
		}
		ui.PrintSuccess("uv installed successfully")
	}()

	// Install segger-jlink
	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.commandExists("JLinkExe") {
			ui.PrintSuccess("segger-jlink already installed")
			return
		}

		ui.PrintInfo("Installing segger-jlink (this may take a few minutes)...")
		if err := d.runBrewInstall("segger-jlink", true); err != nil {
			errChan <- fmt.Errorf("failed to install segger-jlink: %w", err)
			return
		}
		ui.PrintSuccess("segger-jlink installed successfully")
	}()

	// Wait for both installations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckJLinkProbe checks if a J-Link probe is connected
func (d *DarwinInstaller) CheckJLinkProbe() bool {
	// Use ioreg (fast, works on macOS 10.5+)
	cmd := exec.Command("ioreg", "-p", "IOUSB", "-l", "-w", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	outputStr := strings.ToLower(string(output))
	// Look for SEGGER
	return strings.Contains(outputStr, "segger")
}

// FlashBoard flashes the specified board using uvx
func (d *DarwinInstaller) FlashBoard(orgID, apiToken, board string) (string, error) {
	ui.PrintInfo(fmt.Sprintf("Flashing board: %s", board))
	ui.PrintInfo("This may take 10-15 seconds...")

	// Find the uv binary location
	uvPath, err := exec.LookPath("uv")
	if err != nil {
		return "", fmt.Errorf("uv not found in PATH: %w", err)
	}

	if IsDebugMode() {
		ui.PrintDebug(fmt.Sprintf("Using uv at: %s", uvPath))
		ui.PrintDebug(fmt.Sprintf("Org ID: %s", orgID))
		if len(apiToken) > 11 {
			ui.PrintDebug(fmt.Sprintf("API Token: %s...%s (length: %d)", apiToken[:7], apiToken[len(apiToken)-4:], len(apiToken)))
		} else {
			ui.PrintDebug(fmt.Sprintf("API Token length: %d", len(apiToken)))
		}
	}

	// Build the command - use 'uv tool run' instead of 'uvx'
	cmd := exec.Command(uvPath, "tool", "run", "--from", "pyhubbledemo", "hubbledemo", "flash", board, "-o", orgID, "-t", apiToken)

	if IsDebugMode() {
		// Show the command without the token for security
		cmdStr := fmt.Sprintf("%s tool run --from pyhubbledemo hubbledemo flash %s -o %s -t [REDACTED]", uvPath, board, orgID)
		ui.PrintDebug(fmt.Sprintf("Command: %s", cmdStr))
	}

	// Suppress Python warnings (SyntaxWarning, DeprecationWarning, etc.)
	cmd.Env = append(os.Environ(), "PYTHONWARNINGS=ignore")

	// Create pipes for real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start flash command: %w", err)
	}

	// Channel to capture device name from output
	deviceNameChan := make(chan string, 1)

	// Read and display output in real-time, capturing device name
	go d.streamOutputAndCaptureDeviceName(stdout, deviceNameChan)
	go d.streamOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("flash command failed: %w", err)
	}

	// Get device name from channel (with default if not found)
	var deviceName string
	select {
	case deviceName = <-deviceNameChan:
	default:
		deviceName = "your-device"
	}

	ui.PrintSuccess(fmt.Sprintf("Board %s flashed successfully!", board))
	return deviceName, nil
}

// Verify verifies the installation was successful
func (d *DarwinInstaller) Verify() error {
	// Check that all required tools are available
	tools := []string{"brew", "uv", "JLinkExe"}

	for _, tool := range tools {
		if !d.commandExists(tool) {
			return fmt.Errorf("verification failed: %s not found", tool)
		}
	}

	ui.PrintSuccess("Installation verified - all tools present")
	return nil
}

// Helper functions

// commandExists checks if a command is available in PATH
func (d *DarwinInstaller) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// setupBrewPath adds Homebrew to PATH for the current process
func (d *DarwinInstaller) setupBrewPath() error {
	// Detect Homebrew installation path based on architecture
	// Apple Silicon: /opt/homebrew
	// Intel: /usr/local
	var brewPath string
	if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
		brewPath = "/opt/homebrew/bin"
	} else if _, err := os.Stat("/usr/local/bin/brew"); err == nil {
		brewPath = "/usr/local/bin"
	} else {
		return fmt.Errorf("brew not found in expected locations")
	}

	// Update PATH for this process
	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, brewPath) {
		newPath := brewPath + ":" + currentPath
		os.Setenv("PATH", newPath)

		if IsDebugMode() {
			ui.PrintDebug(fmt.Sprintf("Added %s to PATH", brewPath))
		}
	}

	return nil
}

// runBrewInstall runs a brew install command
func (d *DarwinInstaller) runBrewInstall(pkg string, showOutput bool) error {
	cmd := exec.Command("brew", "install", pkg)

	// Show output if requested or in debug mode
	if showOutput || IsDebugMode() {
		if IsDebugMode() {
			ui.PrintDebug(fmt.Sprintf("Running: brew install %s", pkg))
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// streamOutput streams command output line by line
func (d *DarwinInstaller) streamOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println("  " + scanner.Text())
	}
}

// streamOutputAndCaptureDeviceName streams output and captures the device name
func (d *DarwinInstaller) streamOutputAndCaptureDeviceName(pipe io.ReadCloser, deviceNameChan chan<- string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("  " + line)

		// Look for device name in the output
		// Pattern: [INFO] No name supplied. Naming device "device-name"
		if strings.Contains(line, "Naming device") {
			// Find the quoted device name
			startQuote := strings.Index(line, "\"")
			if startQuote != -1 {
				endQuote := strings.Index(line[startQuote+1:], "\"")
				if endQuote != -1 {
					deviceName := line[startQuote+1 : startQuote+1+endQuote]
					if deviceName != "" {
						select {
						case deviceNameChan <- deviceName:
						default:
						}
					}
				}
			}
		}
	}
}
