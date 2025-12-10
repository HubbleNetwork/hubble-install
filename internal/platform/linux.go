package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// PackageManager represents the type of package manager
type PackageManager int

const (
	PackageManagerUnknown PackageManager = iota
	PackageManagerAPT                    // Debian, Ubuntu, etc.
	PackageManagerYUM                    // RHEL, CentOS (older)
	PackageManagerDNF                    // Fedora, RHEL 8+
)

// LinuxInstaller implements the Installer interface for Linux
type LinuxInstaller struct {
	pkgManager PackageManager
}

// NewLinuxInstaller creates a new Linux installer
func NewLinuxInstaller() *LinuxInstaller {
	return &LinuxInstaller{
		pkgManager: detectPackageManager(),
	}
}

// Name returns the platform name
func (l *LinuxInstaller) Name() string {
	return "Linux"
}

// ensureSudoAccess validates sudo access upfront to avoid multiple password prompts
func (l *LinuxInstaller) ensureSudoAccess() error {
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

// CheckPrerequisites checks for missing dependencies based on required deps
func (l *LinuxInstaller) CheckPrerequisites(requiredDeps []string) ([]MissingDependency, error) {
	var missing []MissingDependency

	// Check if package manager is supported
	if l.pkgManager == PackageManagerUnknown {
		return nil, fmt.Errorf("unsupported Linux distribution - only apt, dnf, and yum are supported")
	}

	// Check each required dependency
	for _, dep := range requiredDeps {
		switch dep {
		case "uv":
			if !l.commandExists("uv") {
				missing = append(missing, MissingDependency{
					Name:   "uv",
					Status: "Not installed",
				})
			}
		case "segger-jlink":
			// Check for SEGGER J-Link (must be installed manually on Linux)
			if !l.commandExists("JLinkExe") {
				fmt.Println("") // blank line for readability
				ui.PrintError("SEGGER J-Link was not found")
				ui.PrintInfo("Due to license requirements, it must be downloaded manually from:")
				ui.PrintInfo("  https://www.segger.com/downloads/jlink/")
				fmt.Println("") // blank line
				ui.PrintInfo("After downloading, install with:")

				switch l.pkgManager {
				case PackageManagerAPT:
					ui.PrintInfo("  sudo dpkg -i JLink_Linux_*.deb")
				case PackageManagerDNF:
					ui.PrintInfo("  sudo dnf install JLink_Linux_*.rpm")
				case PackageManagerYUM:
					ui.PrintInfo("  sudo yum install JLink_Linux_*.rpm")
				default:
					ui.PrintInfo("  tar xzf JLink_Linux_*.tgz -C ~/opt/SEGGER")
					ui.PrintInfo("  sudo cp ~/opt/SEGGER/JLink*/99-jlink.rules /etc/udev/rules.d/")
				}

				fmt.Println("") // blank line
				return nil, fmt.Errorf("J-Link must be installed before running this installer")
			}
		}
	}

	return missing, nil
}

// InstallPackageManager is not needed for Linux (uv and jlink use direct installers)
func (l *LinuxInstaller) InstallPackageManager() error {
	// Both uv (astral.sh) and jlink (SEGGER) use their own installers
	// No package manager operations needed
	return nil
}

// InstallDependencies installs the specified dependencies
func (l *LinuxInstaller) InstallDependencies(deps []string) error {
	for _, dep := range deps {
		switch dep {
		case "uv":
			// Install uv (must be installed via astral.sh installer)
			if !l.commandExists("uv") {
				ui.PrintInfo("Installing uv from astral.sh...")
				if err := l.installUV(); err != nil {
					return fmt.Errorf("failed to install uv: %w", err)
				}
				ui.PrintSuccess("uv installed successfully")
			} else {
				ui.PrintSuccess("uv already installed")
			}
		case "segger-jlink":
			// J-Link must be installed manually on Linux - verified in CheckPrerequisites
			if l.commandExists("JLinkExe") {
				ui.PrintSuccess("segger-jlink already installed")
			}
		}
	}

	return nil
}

// CleanDependencies removes uv and segger-jlink
func (l *LinuxInstaller) CleanDependencies() error {
	var errors []string

	// Ensure we have sudo access
	if err := l.ensureSudoAccess(); err != nil {
		return err
	}

	// Uninstall uv if present (installed via astral.sh, not package manager)
	if l.commandExists("uv") {
		ui.PrintInfo("Removing uv...")

		homeDir := os.Getenv("HOME")
		uvBinary := filepath.Join(homeDir, ".cargo", "bin", "uv")

		if _, err := os.Stat(uvBinary); err == nil {
			if err := os.Remove(uvBinary); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove uv binary: %v", err))
			} else {
				ui.PrintSuccess("uv removed")
			}
		}

		// Remove uv cache
		uvCache := filepath.Join(homeDir, ".cache", "uv")
		if _, err := os.Stat(uvCache); err == nil {
			if IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Removing cache: %s", uvCache))
			}
			if err := os.RemoveAll(uvCache); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove uv cache: %v", err))
			}
		}
	}

	// Uninstall segger-jlink if present
	if l.commandExists("JLinkExe") {
		ui.PrintInfo("Removing segger-jlink...")

		// Check if installed via package manager (DEB/RPM)
		var pkgInstalled bool
		switch l.pkgManager {
		case PackageManagerAPT:
			checkCmd := exec.Command("dpkg", "-l", "jlink")
			pkgInstalled = checkCmd.Run() == nil
		case PackageManagerDNF, PackageManagerYUM:
			checkCmd := exec.Command("rpm", "-q", "jlink")
			pkgInstalled = checkCmd.Run() == nil
		}

		if pkgInstalled {
			// Remove via package manager
			if err := l.removeJLinkPackage(); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove segger-jlink package: %v", err))
			} else {
				ui.PrintSuccess("segger-jlink package removed")
			}
		} else {
			// Remove TGZ installation
			homeDir := os.Getenv("HOME")
			jlinkDir := filepath.Join(homeDir, "opt/SEGGER")

			if _, err := os.Stat(jlinkDir); err == nil {
				ui.PrintInfo(fmt.Sprintf("Removing %s...", jlinkDir))
				if err := os.RemoveAll(jlinkDir); err != nil {
					errors = append(errors, fmt.Sprintf("failed to remove J-Link directory: %v", err))
				} else {
					ui.PrintSuccess("segger-jlink removed")
				}

				// Remove UDEV rules
				udevRules := "/etc/udev/rules.d/99-jlink.rules"
				if _, err := os.Stat(udevRules); err == nil {
					rmCmd := exec.Command("sudo", "rm", udevRules)
					if err := rmCmd.Run(); err != nil {
						ui.PrintWarning("Failed to remove UDEV rules - you may need to remove manually")
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %v", errors)
	}

	return nil
}

// installUV installs uv using the official astral.sh installer
func (l *LinuxInstaller) installUV() error {
	// Download and run the uv installer script
	cmd := exec.Command("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv installation failed: %w", err)
	}

	// Add uv to PATH for current process
	// The installer puts it in ~/.cargo/bin
	homeDir := os.Getenv("HOME")
	cargoPath := filepath.Join(homeDir, ".cargo", "bin")

	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, cargoPath) {
		os.Setenv("PATH", cargoPath+":"+currentPath)

		if IsDebugMode() {
			ui.PrintDebug(fmt.Sprintf("Added %s to PATH", cargoPath))
		}
	}

	return nil
}

// CheckJLinkProbe checks if a J-Link probe is connected
func (l *LinuxInstaller) CheckJLinkProbe() bool {
	// Use lsusb to check for SEGGER devices
	cmd := exec.Command("lsusb")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	outputStr := strings.ToLower(string(output))
	// Look for SEGGER
	return strings.Contains(outputStr, "segger")
}

// FlashBoard flashes the specified board using uvx (for J-Link boards)
func (l *LinuxInstaller) FlashBoard(orgID, apiToken, board string) (*FlashResult, error) {
	ui.PrintInfo(fmt.Sprintf("Flashing board: %s", board))
	ui.PrintInfo("This may take 10-15 seconds...")

	// Find the uv binary location
	uvPath, err := exec.LookPath("uv")
	if err != nil {
		return nil, fmt.Errorf("uv not found in PATH: %w", err)
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
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start flash command: %w", err)
	}

	// Channel to capture device name from output
	deviceNameChan := make(chan string, 1)

	// Read and display output in real-time, capturing device name
	go l.streamOutputAndCaptureDeviceName(stdout, deviceNameChan)
	go l.streamOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("flash command failed: %w", err)
	}

	// Get device name from channel (with default if not found)
	var deviceName string
	select {
	case deviceName = <-deviceNameChan:
	default:
		deviceName = "your-device"
	}

	ui.PrintSuccess(fmt.Sprintf("Board %s flashed successfully!", board))
	return &FlashResult{DeviceName: deviceName}, nil
}

// GenerateHexFile generates a hex file for Uniflash boards (TI)
func (l *LinuxInstaller) GenerateHexFile(orgID, apiToken, board string) (*FlashResult, error) {
	ui.PrintInfo(fmt.Sprintf("Generating hex file for board: %s", board))
	ui.PrintInfo("This may take a few seconds...")

	// Find the uv binary location
	uvPath, err := exec.LookPath("uv")
	if err != nil {
		return nil, fmt.Errorf("uv not found in PATH: %w", err)
	}

	if IsDebugMode() {
		ui.PrintDebug(fmt.Sprintf("Using uv at: %s", uvPath))
	}

	// Build the command
	cmd := exec.Command(uvPath, "tool", "run", "--from", "pyhubbledemo", "hubbledemo", "flash", board, "-o", orgID, "-t", apiToken)

	if IsDebugMode() {
		cmdStr := fmt.Sprintf("%s tool run --from pyhubbledemo hubbledemo flash %s -o %s -t [REDACTED]", uvPath, board, orgID)
		ui.PrintDebug(fmt.Sprintf("Command: %s", cmdStr))
	}

	cmd.Env = append(os.Environ(), "PYTHONWARNINGS=ignore")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	hexPathChan := make(chan string, 1)
	go l.streamOutputAndCaptureHexPath(stdout, hexPathChan)
	go l.streamOutput(stderr)

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	var hexPath string
	select {
	case hexPath = <-hexPathChan:
	default:
		homeDir := os.Getenv("HOME")
		hexPath = filepath.Join(homeDir, ".hubble", board+".hex")
	}

	ui.PrintSuccess("Hex file generated successfully!")
	return &FlashResult{HexFilePath: hexPath}, nil
}

// Verify verifies the installation was successful for the given dependencies
func (l *LinuxInstaller) Verify(deps []string) error {
	for _, dep := range deps {
		switch dep {
		case "uv":
			if !l.commandExists("uv") {
				return fmt.Errorf("verification failed: uv not found")
			}
		case "segger-jlink":
			if !l.commandExists("JLinkExe") {
				return fmt.Errorf("verification failed: JLinkExe not found")
			}
		}
	}

	ui.PrintSuccess("Installation verified - all tools present")
	return nil
}

// Helper functions

// detectPackageManager detects which package manager is available
func detectPackageManager() PackageManager {
	if commandExistsGlobal("apt-get") {
		return PackageManagerAPT
	}
	if commandExistsGlobal("dnf") {
		return PackageManagerDNF
	}
	if commandExistsGlobal("yum") {
		return PackageManagerYUM
	}
	return PackageManagerUnknown
}

// commandExists checks if a command is available in PATH
func (l *LinuxInstaller) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// commandExistsGlobal checks if a command is available (global function for init)
func commandExistsGlobal(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// installPackage installs a package using the detected package manager
func (l *LinuxInstaller) installPackage(pkg string, showOutput bool) error {
	var cmd *exec.Cmd

	switch l.pkgManager {
	case PackageManagerAPT:
		cmd = exec.Command("sudo", "apt-get", "install", "-y", pkg)
	case PackageManagerDNF:
		cmd = exec.Command("sudo", "dnf", "install", "-y", pkg)
	case PackageManagerYUM:
		cmd = exec.Command("sudo", "yum", "install", "-y", pkg)
	default:
		return fmt.Errorf("unsupported package manager")
	}

	// Show output if requested or in debug mode
	if showOutput || IsDebugMode() {
		if IsDebugMode() {
			ui.PrintDebug(fmt.Sprintf("Running package install: %s", pkg))
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// removePackage removes a package using the detected package manager
func (l *LinuxInstaller) removePackage(pkg string) error {
	var cmd *exec.Cmd

	switch l.pkgManager {
	case PackageManagerAPT:
		cmd = exec.Command("sudo", "apt-get", "remove", "-y", pkg)
	case PackageManagerDNF:
		cmd = exec.Command("sudo", "dnf", "remove", "-y", pkg)
	case PackageManagerYUM:
		cmd = exec.Command("sudo", "yum", "remove", "-y", pkg)
	default:
		return fmt.Errorf("unsupported package manager")
	}

	if IsDebugMode() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// removeJLinkPackage removes J-Link if installed via DEB/RPM package
func (l *LinuxInstaller) removeJLinkPackage() error {
	var cmd *exec.Cmd

	switch l.pkgManager {
	case PackageManagerAPT:
		cmd = exec.Command("sudo", "dpkg", "-r", "jlink")
	case PackageManagerDNF:
		cmd = exec.Command("sudo", "dnf", "remove", "-y", "jlink")
	case PackageManagerYUM:
		cmd = exec.Command("sudo", "yum", "remove", "-y", "jlink")
	default:
		return fmt.Errorf("unsupported package manager")
	}

	if IsDebugMode() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// streamOutput streams command output line by line
func (l *LinuxInstaller) streamOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println("  " + scanner.Text())
	}
}

// streamOutputAndCaptureDeviceName streams output and captures the device name
func (l *LinuxInstaller) streamOutputAndCaptureDeviceName(pipe io.ReadCloser, deviceNameChan chan<- string) {
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

// streamOutputAndCaptureHexPath streams output and captures the hex file path
func (l *LinuxInstaller) streamOutputAndCaptureHexPath(pipe io.ReadCloser, hexPathChan chan<- string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("  " + line)

		// Look for hex file path in the output
		// Pattern: Hex file written to "/path/to/file.hex"
		if strings.Contains(line, ".hex") {
			// Extract quoted path
			startQuote := strings.Index(line, "\"")
			if startQuote != -1 {
				endQuote := strings.Index(line[startQuote+1:], "\"")
				if endQuote != -1 {
					hexPath := line[startQuote+1 : startQuote+1+endQuote]
					if strings.HasSuffix(hexPath, ".hex") {
						select {
						case hexPathChan <- hexPath:
						default:
						}
					}
				}
			}
		}
	}
}
