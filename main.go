package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/HubbleNetwork/hubble-install/internal/boards"
	"github.com/HubbleNetwork/hubble-install/internal/config"
	"github.com/HubbleNetwork/hubble-install/internal/platform"
	"github.com/HubbleNetwork/hubble-install/internal/ui"
	"github.com/rudderlabs/analytics-go/v4"
)

// Analytics configuration
//
// These metrics are completely anonymous (UserId is always "anonymous") and are used
// solely for quality assurance and bug tracking. We track:
// - Which step users reach in the installer flow
// - Error messages when installations fail
// - Platform/architecture distribution
// - Installation success rates and duration
//
// No personally identifiable information is collected.
//
// Users can opt out of analytics by:
// - Setting environment variable: HUBBLE_NO_ANALYTICS=1
// - Adding a=false to the install URL query string
const (
	rudderWriteKey  = "39Ge5rWJeNuA7s0HRg7jZEdyScj"
	rudderDataPlane = "https://hubblejohsbrmt.dataplane.rudderstack.com"
)

// analyticsEnabled checks if analytics should be enabled
// Users can opt out via HUBBLE_NO_ANALYTICS env var or a=false argument
func analyticsEnabled() bool {
	// Check environment variable
	if os.Getenv("HUBBLE_NO_ANALYTICS") != "" {
		return false
	}

	// Check command line arguments for a=false (opt-out) or a=true (opt-in)
	for _, arg := range os.Args[1:] {
		if arg == "a=false" || arg == "--no-analytics" {
			return false
		}
	}

	return true
}

// Analytics event names
const (
	eventStart   = "hubble-install-start"
	eventStep    = "hubble-install-step"
	eventError   = "hubble-install-error"
	eventSuccess = "hubble-install-success"
)

// analyticsClient wraps the Rudderstack client for tracking
type analyticsClient struct {
	client   analytics.Client
	platform string
	arch     string
	enabled  bool
}

// newAnalyticsClient creates a new analytics client (or a no-op client if disabled)
func newAnalyticsClient() *analyticsClient {
	enabled := analyticsEnabled()
	var client analytics.Client
	if enabled {
		client = analytics.New(rudderWriteKey, rudderDataPlane)
	}
	return &analyticsClient{
		client:   client,
		platform: runtime.GOOS,
		arch:     runtime.GOARCH,
		enabled:  enabled,
	}
}

// Close flushes and closes the analytics client
func (a *analyticsClient) Close() {
	if a.enabled && a.client != nil {
		a.client.Close()
	}
}

// baseProperties returns common properties for all events
func (a *analyticsClient) baseProperties() analytics.Properties {
	return analytics.NewProperties().
		Set("platform", a.platform).
		Set("arch", a.arch)
}

// TrackStart tracks the installer start event
func (a *analyticsClient) TrackStart() {
	if !a.enabled {
		return
	}
	a.client.Enqueue(analytics.Track{
		UserId:     "anonymous",
		Event:      eventStart,
		Properties: a.baseProperties(),
	})
}

// TrackStep tracks a step event
func (a *analyticsClient) TrackStep(step int, stepName string) {
	if !a.enabled {
		return
	}
	a.client.Enqueue(analytics.Track{
		UserId: "anonymous",
		Event:  eventStep,
		Properties: a.baseProperties().
			Set("step", step).
			Set("step_name", stepName),
	})
}

// TrackError tracks an error event
func (a *analyticsClient) TrackError(step int, stepName string, errMsg string) {
	if !a.enabled {
		return
	}
	// Sanitize error message to remove potential PII (home directory paths)
	sanitizedErr := sanitizeErrorMessage(errMsg)
	a.client.Enqueue(analytics.Track{
		UserId: "anonymous",
		Event:  eventError,
		Properties: a.baseProperties().
			Set("step", step).
			Set("step_name", stepName).
			Set("error", sanitizedErr),
	})
}

// sanitizeErrorMessage removes potential PII from error messages
func sanitizeErrorMessage(errMsg string) string {
	// Replace home directory paths with ~
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		errMsg = strings.ReplaceAll(errMsg, homeDir, "~")
	}
	// Also handle common patterns like /Users/<username> or /home/<username>
	// by replacing anything that looks like a home path
	if idx := strings.Index(errMsg, "/Users/"); idx != -1 {
		end := strings.IndexAny(errMsg[idx+7:], "/ ")
		if end != -1 {
			errMsg = errMsg[:idx] + "~" + errMsg[idx+7+end:]
		}
	}
	if idx := strings.Index(errMsg, "/home/"); idx != -1 {
		end := strings.IndexAny(errMsg[idx+6:], "/ ")
		if end != -1 {
			errMsg = errMsg[:idx] + "~" + errMsg[idx+6+end:]
		}
	}
	return errMsg
}

// TrackSuccess tracks a successful completion
func (a *analyticsClient) TrackSuccess(durationSecs float64, board string) {
	if !a.enabled {
		return
	}
	a.client.Enqueue(analytics.Track{
		UserId: "anonymous",
		Event:  eventSuccess,
		Properties: a.baseProperties().
			Set("duration_seconds", durationSecs).
			Set("board", board),
	})
}

func main() {
	// Initialize analytics
	tracker := newAnalyticsClient()
	defer tracker.Close()

	// Track installer start
	tracker.TrackStart()

	// Helper to exit with error tracking
	exitWithError := func(step int, stepName string, errMsg string, code int) {
		tracker.TrackError(step, stepName, errMsg)
		tracker.Close()
		os.Exit(code)
	}

	// Print welcome banner
	ui.PrintBanner()
	fmt.Println()

	// Show what will happen
	ui.PrintInfo("This installer will:")
	fmt.Println("  • Confirm your developer board model")
	fmt.Println("  • Check for and install required dependencies")
	fmt.Println("  • Configure your Hubble credentials")
	fmt.Println("  • Register your board to your organization, and give it a name")
	fmt.Println("  • Provision your board, or generate a hex file for you to flash")
	fmt.Println()

	// Prompt user to continue
	if !ui.PromptYesNo("Ready to install?", true) {
		ui.PrintWarning("Installation cancelled")
		tracker.TrackError(0, "welcome", "user_cancelled")
		tracker.Close()
		os.Exit(0)
	}
	fmt.Println()

	// Start timer for the installation
	startTime := time.Now()

	// Detect platform
	installer, err := platform.GetInstaller()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Platform detection failed: %v", err))
		exitWithError(0, "platform_detection", err.Error(), 1)
	}

	// Check for pending reboot (especially important on Windows)
	if err := installer.CheckPendingReboot(); err != nil {
		fmt.Println()
		ui.PrintWarning("═══════════════════════════════════════════════════════════════")
		ui.PrintWarning("  SYSTEM REBOOT REQUIRED")
		ui.PrintWarning("═══════════════════════════════════════════════════════════════")
		fmt.Println()
		ui.PrintWarning("A previous installation requires a system reboot before continuing.")
		ui.PrintInfo(fmt.Sprintf("Reason: %v", err))
		fmt.Println()
		ui.PrintInfo("Please reboot your computer and run this installer again.")
		fmt.Println()
		exitWithError(0, "reboot_check", "reboot_required", 2)
	}

	// Fetch the supported board list (the source of truth is md.json in
	// hubble-tldm). This needs network, but so does the rest of the install, so
	// fail early with a clear message rather than continuing with no boards.
	availableBoards, err := boards.FetchBoards()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Could not load the supported board list: %v", err))
		exitWithError(1, "fetch_boards", err.Error(), 1)
	}

	// =========================================================================
	// Step 1: Get credentials (may include pre-configured board)
	// =========================================================================
	currentStep := 1
	totalSteps := 0
	stepName := "credentials"
	tracker.TrackStep(currentStep, stepName)
	ui.PrintStep("Configuring credentials", currentStep, totalSteps)

	cfg, preConfigured, err := config.PromptForConfig(availableBoards)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Configuration failed: %v", err))
		exitWithError(currentStep, stepName, err.Error(), 1)
	}

	if preConfigured {
		fmt.Println()
		ui.PrintSuccess("We've handled your setup details")
		fmt.Println()
		ui.PrintInfo("We've pre-filled your credentials for this command.")
		fmt.Println()
		ui.PrintInfo("Your Hubble Org ID and API Token are used to register your board to your organization.")
		fmt.Println()
	}

	// =========================================================================
	// Step 2: Select board (if not pre-configured)
	// =========================================================================
	currentStep++
	stepName = "board_selection"
	tracker.TrackStep(currentStep, stepName)
	ui.PrintStep("Selecting developer board", currentStep, totalSteps)

	var selectedBoard boards.Board
	if cfg.Board != "" {
		// Board was pre-configured via credentials
		board, err := boards.GetBoard(availableBoards, cfg.Board)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Invalid pre-configured board: %v", err))
			exitWithError(currentStep, stepName, err.Error(), 1)
		}
		selectedBoard = *board
		ui.PrintSuccess(fmt.Sprintf("Using pre-configured board: %s", selectedBoard.Name))
	} else {
		// Prompt user to select a board
		boardOptions := make([]string, len(availableBoards))
		for i, board := range availableBoards {
			boardOptions[i] = board.Name
		}

		selectedIndex := ui.PromptChoice("Available developer boards:", boardOptions)
		selectedBoard = availableBoards[selectedIndex]
		cfg.Board = selectedBoard.ID

		ui.PrintSuccess(fmt.Sprintf("Selected: %s", selectedBoard.Name))
	}

	fmt.Println()
	if selectedBoard.RequiresJLink() {
		ui.PrintInfo("This board is flashed directly via SEGGER J-Link.")
		ui.PrintWarning("Make sure your board is connected via USB with a data-capable cable.")
	} else {
		ui.PrintInfo("A firmware hex file will be generated for you to flash onto the board.")
	}
	fmt.Println()

	// =========================================================================
	// Step 3: Check prerequisites (based on selected board)
	// =========================================================================
	currentStep++
	stepName = "prerequisites"
	tracker.TrackStep(currentStep, stepName)
	ui.PrintStep("Checking prerequisites", currentStep, totalSteps)

	requiredDeps := selectedBoard.GetDependencies()
	missing, err := installer.CheckPrerequisites(requiredDeps)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Prerequisites check failed: %v", err))
		exitWithError(currentStep, stepName, err.Error(), 1)
	}

	totalSteps = 4
	if len(missing) > 0 {
		totalSteps++
	}

	if len(missing) > 0 {
		ui.PrintWarning("Missing dependencies detected:")
		for _, dep := range missing {
			fmt.Printf("  • %s: %s\n", dep.Name, dep.Status)
		}
		fmt.Println()

		if !ui.PromptYesNo("Would you like to install missing dependencies?", true) {
			ui.PrintError("Cannot proceed without dependencies")
			exitWithError(currentStep, stepName, "user_declined_dependencies", 1)
		}
	} else {
		ui.PrintSuccess("All prerequisites satisfied")
	}

	// =========================================================================
	// Step 4: Install dependencies (only if needed)
	// =========================================================================
	if len(missing) > 0 {
		currentStep++
		stepName = "dependencies"
		tracker.TrackStep(currentStep, stepName)
		ui.PrintStep("Installing dependencies", currentStep, totalSteps)

		// Check if we need to install package manager first
		needsPackageManager := false
		for _, dep := range missing {
			if dep.Name == "Homebrew" {
				needsPackageManager = true
				break
			}
		}

		if needsPackageManager {
			if err := installer.InstallPackageManager(); err != nil {
				ui.PrintError(fmt.Sprintf("Package manager installation failed: %v", err))
				exitWithError(currentStep, stepName, err.Error(), 1)
			}
		}

		// Install board-specific dependencies
		if err := installer.InstallDependencies(requiredDeps); err != nil {
			// Check if this is a reboot required error
			if strings.Contains(err.Error(), "requires a system reboot") || strings.Contains(err.Error(), "RebootRequired") {
				fmt.Println()
				ui.PrintWarning("═══════════════════════════════════════════════════════════════")
				ui.PrintWarning("  SYSTEM REBOOT REQUIRED")
				ui.PrintWarning("═══════════════════════════════════════════════════════════════")
				fmt.Println()
				ui.PrintSuccess("Dependencies were installed successfully!")
				fmt.Println()
				ui.PrintWarning("However, system components were updated that require a reboot")
				ui.PrintWarning("before you can continue.")
				fmt.Println()
				ui.PrintInfo("What to do next:")
				ui.PrintInfo("  1. Reboot your computer")
				ui.PrintInfo("  2. Run this installer again after rebooting")
				ui.PrintInfo("  3. The installer will detect what's already installed and continue")
				fmt.Println()
				ui.PrintInfo("Note: If PowerShell doesn't work after reboot, use Command Prompt (cmd.exe)")
				fmt.Println()
				exitWithError(currentStep, stepName, "reboot_required", 2)
			}
			ui.PrintError(fmt.Sprintf("Dependency installation failed: %v", err))
			exitWithError(currentStep, stepName, err.Error(), 1)
		}

		ui.PrintSuccess("All dependencies installed")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		ui.PrintError(fmt.Sprintf("Invalid configuration: %v", err))
		exitWithError(currentStep, "validation", err.Error(), 1)
	}

	// =========================================================================
	// Final Step: Flash board or generate hex file
	// =========================================================================
	currentStep++
	stepName = "flash"
	tracker.TrackStep(currentStep, stepName)

	if selectedBoard.RequiresJLink() {
		// J-Link path: Direct flash
		if !ui.PromptYesNo(fmt.Sprintf("Would you like to flash your %s now?", selectedBoard.Name), true) {
			ui.PrintWarning("Flashing skipped. You can flash later using:")
			fmt.Printf("  uv tool run --from pyhubbledemo hubbledemo flash %s -o %s -t <your_token>\n", cfg.Board, cfg.OrgID)
			tracker.TrackError(currentStep, stepName, "user_skipped_flash")
			tracker.Close()
			os.Exit(0)
		}

		// Prompt for optional device name
		deviceName := ui.PromptOptionalInput("What should the device name be?")

		ui.PrintStep("Flashing board", currentStep, totalSteps)
		result, err := installer.FlashBoard(cfg.OrgID, cfg.APIToken, cfg.Board, deviceName)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Board flashing failed: %v", err))
			exitWithError(currentStep, stepName, err.Error(), 1)
		}

		// Track success and print J-Link completion banner
		duration := time.Since(startTime)
		tracker.TrackSuccess(duration.Seconds(), cfg.Board)
		ui.PrintCompletionBanner(duration, cfg.OrgID, cfg.APIToken, result.DeviceName)

	} else {
		// Uniflash path: Generate hex file
		if !ui.PromptYesNo(fmt.Sprintf("Would you like to generate the hex file for your %s now?", selectedBoard.Name), true) {
			ui.PrintWarning("Hex generation skipped. You can generate later using:")
			fmt.Printf("  uv tool run --from pyhubbledemo hubbledemo flash %s -o %s -t <your_token>\n", cfg.Board, cfg.OrgID)
			tracker.TrackError(currentStep, stepName, "user_skipped_hex")
			tracker.Close()
			os.Exit(0)
		}

		// Prompt for optional device name
		deviceName := ui.PromptOptionalInput("What should the device name be?")

		ui.PrintStep("Generating hex file", currentStep, totalSteps)
		result, err := installer.GenerateHexFile(cfg.OrgID, cfg.APIToken, cfg.Board, deviceName)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Hex file generation failed: %v", err))
			exitWithError(currentStep, stepName, err.Error(), 1)
		}

		// Track success and print Uniflash completion banner
		duration := time.Since(startTime)
		tracker.TrackSuccess(duration.Seconds(), cfg.Board)
		ui.PrintUniflashCompletionBanner(duration, result.HexFilePath, selectedBoard.Name, deviceName)
	}

	os.Exit(0)
}
