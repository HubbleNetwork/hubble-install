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

// CheckPrerequisites checks for missing dependencies
func (l *LinuxInstaller) CheckPrerequisites() ([]MissingDependency, error) {
	var missing []MissingDependency

	// Check if package manager is supported
	if l.pkgManager == PackageManagerUnknown {
		return nil, fmt.Errorf("unsupported Linux distribution - only apt, dnf, and yum are supported")
	}

	// Check for uv
	if !l.commandExists("uv") {
		missing = append(missing, MissingDependency{
			Name:   "uv",
			Status: "Not installed",
		})
	}

	// Check for JLink (from segger-jlink)
	if !l.commandExists("JLinkExe") {
		missing = append(missing, MissingDependency{
			Name:   "segger-jlink",
			Status: "Not installed",
		})
	}

	return missing, nil
}

// InstallPackageManager is not needed for Linux (uv and jlink use direct installers)
func (l *LinuxInstaller) InstallPackageManager() error {
	// Both uv (astral.sh) and jlink (SEGGER) use their own installers
	// No package manager operations needed
	return nil
}

// InstallDependencies installs uv and checks for segger-jlink
func (l *LinuxInstaller) InstallDependencies() error {
	// Install uv (must be installed via astral.sh installer)
	if !l.commandExists("uv") {
		ui.PrintInfo("Installing dependencies for uv...")
		if err := l.installUV(); err != nil {
			return fmt.Errorf("failed to install uv: %w", err)
		}
		ui.PrintSuccess("uv installed successfully")
	} else {
		ui.PrintSuccess("uv already installed")
	}

	// Check for segger-jlink (cannot auto-install - requires manual download from SEGGER)
	ui.PrintInfo("Checking for SEGGER J-Link...")

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
		return fmt.Errorf("exiting installer: J-Link is required to continue")
	}

	ui.PrintSuccess("SEGGER J-Link found")
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

// FlashBoard flashes the specified board using uvx
func (l *LinuxInstaller) FlashBoard(orgID, apiToken, board string) error {
	ui.PrintInfo(fmt.Sprintf("Flashing board: %s", board))
	ui.PrintInfo("This may take 10-15 seconds...")

	// Find the uv binary location
	uvPath, err := exec.LookPath("uv")
	if err != nil {
		return fmt.Errorf("uv not found in PATH: %w", err)
	}

	if IsDebugMode() {
		ui.PrintDebug(fmt.Sprintf("Using uv at: %s", uvPath))
	}

	// Build the command - use 'uv tool run' instead of 'uvx'
	cmd := exec.Command(uvPath, "tool", "run", "--from", "pyhubbledemo", "hubbledemo", "flash", board, "-o", orgID, "-t", apiToken)

	// Suppress Python warnings (SyntaxWarning, DeprecationWarning, etc.)
	cmd.Env = append(os.Environ(), "PYTHONWARNINGS=ignore")

	// Create pipes for real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start flash command: %w", err)
	}

	// Read and display output in real-time
	go l.streamOutput(stdout)
	go l.streamOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("flash command failed: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Board %s flashed successfully!", board))
	return nil
}

// Verify verifies the installation was successful
func (l *LinuxInstaller) Verify() error {
	// Check that all required tools are available
	tools := []string{"uv", "JLinkExe"}

	for _, tool := range tools {
		if !l.commandExists(tool) {
			return fmt.Errorf("verification failed: %s not found", tool)
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
