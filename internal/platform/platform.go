package platform

import (
	"fmt"
	"runtime"
)

var debugMode bool

// SetDebugMode enables or disables debug mode globally
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	return debugMode
}

// MissingDependency represents a missing system dependency
type MissingDependency struct {
	Name   string
	Status string
}

// FlashResult contains the result of a flash operation
type FlashResult struct {
	DeviceName  string // Device name (for J-Link flash)
	HexFilePath string // Path to generated hex file (for Uniflash)
}

// Installer defines the interface for platform-specific installation
type Installer interface {
	// Name returns the platform name
	Name() string

	// CheckPrerequisites checks for missing dependencies based on required deps
	CheckPrerequisites(requiredDeps []string) ([]MissingDependency, error)

	// InstallPackageManager installs the package manager (e.g., Homebrew)
	InstallPackageManager() error

	// InstallDependencies installs the specified dependencies
	InstallDependencies(deps []string) error

	// CleanDependencies removes uv and segger-jlink and clears Homebrew cache
	CleanDependencies() error

	// FlashBoard flashes the specified board with credentials and returns the result
	FlashBoard(orgID, apiToken, board string) (*FlashResult, error)

	// GenerateHexFile generates a hex file for Uniflash boards and returns the path
	GenerateHexFile(orgID, apiToken, board string) (*FlashResult, error)

	// Verify verifies the installation was successful for the given dependencies
	Verify(deps []string) error
}

// GetInstaller returns the appropriate installer for the current platform
func GetInstaller() (Installer, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewDarwinInstaller(), nil
	case "linux":
		return NewLinuxInstaller(), nil
	case "windows":
		return NewWindowsInstaller(), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
