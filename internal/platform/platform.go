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

// Installer defines the interface for platform-specific installation
type Installer interface {
	// Name returns the platform name
	Name() string

	// CheckPrerequisites checks for missing dependencies
	CheckPrerequisites() ([]MissingDependency, error)

	// InstallPackageManager installs the package manager (e.g., Homebrew)
	InstallPackageManager() error

	// InstallDependencies installs required dependencies (uv, segger-jlink)
	InstallDependencies() error

	// CleanDependencies removes uv and segger-jlink and clears Homebrew cache
	CleanDependencies() error

	// CheckJLinkProbe checks if a J-Link probe is connected
	CheckJLinkProbe() bool

	// FlashBoard flashes the specified board with credentials
	FlashBoard(orgID, apiToken, board string) error

	// Verify verifies the installation was successful
	Verify() error
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
