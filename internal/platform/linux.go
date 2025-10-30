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

// InstallPackageManager ensures package manager is up to date
func (l *LinuxInstaller) InstallPackageManager() error {
	// Package manager is already installed on Linux, just update package lists
	if err := l.ensureSudoAccess(); err != nil {
		return err
	}

	ui.PrintInfo("Updating package lists...")

	var cmd *exec.Cmd
	switch l.pkgManager {
	case PackageManagerAPT:
		cmd = exec.Command("sudo", "apt-get", "update", "-y")
	case PackageManagerDNF:
		cmd = exec.Command("sudo", "dnf", "check-update", "-y")
	case PackageManagerYUM:
		cmd = exec.Command("sudo", "yum", "check-update", "-y")
	default:
		return fmt.Errorf("unsupported package manager")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Note: check-update returns non-zero if updates are available, so we ignore the error
	cmd.Run()

	ui.PrintSuccess("Package lists updated")
	return nil
}

// InstallDependencies installs uv and segger-jlink
func (l *LinuxInstaller) InstallDependencies() error {
	// Ensure we have sudo access
	if err := l.ensureSudoAccess(); err != nil {
		return err
	}

	// Update package lists first
	if err := l.InstallPackageManager(); err != nil {
		return err
	}

	// Install uv and segger-jlink in parallel for speed
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Install uv
	wg.Add(1)
	go func() {
		defer wg.Done()
		if l.commandExists("uv") {
			ui.PrintSuccess("uv already installed")
			return
		}

		ui.PrintInfo("Installing uv...")
		if err := l.installPackage("uv", false); err != nil {
			errChan <- fmt.Errorf("failed to install uv: %w", err)
			return
		}
		ui.PrintSuccess("uv installed successfully")
	}()

	// Install segger-jlink
	wg.Add(1)
	go func() {
		defer wg.Done()
		if l.commandExists("JLinkExe") {
			ui.PrintSuccess("segger-jlink already installed")
			return
		}

		ui.PrintInfo("Installing segger-jlink (this may take a few minutes)...")
		if err := l.installPackage("segger-jlink", true); err != nil {
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

// CleanDependencies removes uv and segger-jlink
func (l *LinuxInstaller) CleanDependencies() error {
	var errors []string

	// Ensure we have sudo access
	if err := l.ensureSudoAccess(); err != nil {
		return err
	}

	// Uninstall uv if present
	if l.commandExists("uv") {
		ui.PrintInfo("Removing uv...")
		if err := l.removePackage("uv"); err != nil {
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
	if l.commandExists("JLinkExe") {
		ui.PrintInfo("Removing segger-jlink...")
		if err := l.removePackage("segger-jlink"); err != nil {
			errors = append(errors, fmt.Sprintf("failed to remove segger-jlink: %v", err))
		} else {
			ui.PrintSuccess("segger-jlink removed")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %v", errors)
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

// streamOutput streams command output line by line
func (l *LinuxInstaller) streamOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println("  " + scanner.Text())
	}
}
