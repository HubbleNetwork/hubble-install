package platform

import (
	"fmt"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// WindowsInstaller implements the Installer interface for Windows
type WindowsInstaller struct{}

// NewWindowsInstaller creates a new Windows installer
func NewWindowsInstaller() *WindowsInstaller {
	return &WindowsInstaller{}
}

// Name returns the platform name
func (w *WindowsInstaller) Name() string {
	return "Windows"
}

// CheckPrerequisites checks for missing dependencies
func (w *WindowsInstaller) CheckPrerequisites() ([]MissingDependency, error) {
	ui.PrintWarning("Windows support coming soon!")
	return nil, fmt.Errorf("windows platform not yet implemented")
}

// InstallPackageManager is not implemented for Windows yet
func (w *WindowsInstaller) InstallPackageManager() error {
	return fmt.Errorf("windows platform not yet implemented")
}

// InstallDependencies is not implemented for Windows yet
func (w *WindowsInstaller) InstallDependencies() error {
	return fmt.Errorf("windows platform not yet implemented")
}

// CleanDependencies is not implemented for Windows yet
func (w *WindowsInstaller) CleanDependencies() error {
	return fmt.Errorf("windows platform not yet implemented")
}

// FlashBoard is not implemented for Windows yet
func (w *WindowsInstaller) FlashBoard(orgID, apiToken, board string) error {
	return fmt.Errorf("windows platform not yet implemented")
}

// Verify is not implemented for Windows yet
func (w *WindowsInstaller) Verify() error {
	return fmt.Errorf("windows platform not yet implemented")
}
