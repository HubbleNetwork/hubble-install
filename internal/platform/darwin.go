package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	ui.PrintInfo("Installing Homebrew...")
	ui.PrintWarning("This may require your password and will take a few minutes")

	// Run the official Homebrew installation script
	cmd := exec.Command("/bin/bash", "-c", `$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
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

		brewErr := cmd.Run()

		// Also try removing uv's standalone installation
		uvPath, _ := exec.LookPath("uv")
		if uvPath != "" {
			if IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Found uv at: %s", uvPath))
			}

			// Remove the binary
			if err := os.Remove(uvPath); err != nil && IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Could not remove %s: %v", uvPath, err))
			}
		}

		// Remove uv data directory
		uvDir := os.ExpandEnv("$HOME/.local/bin/uv")
		if _, err := os.Stat(uvDir); err == nil {
			if IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Removing: %s", uvDir))
			}
			os.Remove(uvDir)
		}

		// Remove uv cache
		uvCache := os.ExpandEnv("$HOME/.cache/uv")
		if _, err := os.Stat(uvCache); err == nil {
			if IsDebugMode() {
				ui.PrintDebug(fmt.Sprintf("Removing cache: %s", uvCache))
			}
			os.RemoveAll(uvCache)
		}

		// Check if uv still exists
		if d.commandExists("uv") {
			errors = append(errors, fmt.Sprintf("failed to completely remove uv (brew error: %v)", brewErr))
			ui.PrintWarning("uv may still be partially installed")
		} else {
			ui.PrintSuccess("uv removed")
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
		if err := d.runBrewInstall("uv"); err != nil {
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

		ui.PrintInfo("Installing segger-jlink...")
		if err := d.runBrewInstall("segger-jlink"); err != nil {
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

// FlashBoard flashes the specified board using uvx
func (d *DarwinInstaller) FlashBoard(orgID, apiToken, board string) error {
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
	go d.streamOutput(stdout)
	go d.streamOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("flash command failed: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Board %s flashed successfully!", board))
	return nil
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

// runBrewInstall runs a brew install command
func (d *DarwinInstaller) runBrewInstall(pkg string) error {
	cmd := exec.Command("brew", "install", pkg)

	// Show output in debug mode, otherwise suppress it
	if IsDebugMode() {
		ui.PrintDebug(fmt.Sprintf("Running: brew install %s", pkg))
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
