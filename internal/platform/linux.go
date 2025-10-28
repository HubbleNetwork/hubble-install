package platform

import (
	"fmt"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// LinuxInstaller implements the Installer interface for Linux
type LinuxInstaller struct{}

// NewLinuxInstaller creates a new Linux installer
func NewLinuxInstaller() *LinuxInstaller {
	return &LinuxInstaller{}
}

// Name returns the platform name
func (l *LinuxInstaller) Name() string {
	return "Linux"
}

// CheckPrerequisites checks for missing dependencies
func (l *LinuxInstaller) CheckPrerequisites() ([]MissingDependency, error) {
	ui.PrintWarning("Linux support coming soon!")
	return nil, fmt.Errorf("linux platform not yet implemented")
}

// InstallPackageManager is not implemented for Linux yet
func (l *LinuxInstaller) InstallPackageManager() error {
	return fmt.Errorf("linux platform not yet implemented")
}

// InstallDependencies is not implemented for Linux yet
func (l *LinuxInstaller) InstallDependencies() error {
	return fmt.Errorf("linux platform not yet implemented")
}

// CleanDependencies is not implemented for Linux yet
func (l *LinuxInstaller) CleanDependencies() error {
	return fmt.Errorf("linux platform not yet implemented")
}

// CheckJLinkProbe checks if a J-Link probe is connected (placeholder)
func (l *LinuxInstaller) CheckJLinkProbe() bool {
	return true // Placeholder - always return true
}

// FlashBoard is not implemented for Linux yet
func (l *LinuxInstaller) FlashBoard(orgID, apiToken, board string) error {
	return fmt.Errorf("linux platform not yet implemented")
}

// Verify is not implemented for Linux yet
func (l *LinuxInstaller) Verify() error {
	return fmt.Errorf("linux platform not yet implemented")
}
