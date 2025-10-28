package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/HubbleNetwork/hubble-install/internal/boards"
	"github.com/HubbleNetwork/hubble-install/internal/config"
	"github.com/HubbleNetwork/hubble-install/internal/platform"
	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

const totalSteps = 6

var (
	cleanFlag bool
	debugFlag bool
)

func main() {
	// Parse command line flags
	flag.BoolVar(&cleanFlag, "clean", false, "Remove existing uv and segger-jlink dependencies and clear Homebrew cache, then exit")
	flag.BoolVar(&debugFlag, "debug", false, "Enable debug mode (reserved for future use)")
	flag.Parse()

	// Set debug mode globally
	platform.SetDebugMode(debugFlag)

	// Print welcome banner
	ui.PrintBanner()

	// Handle clean flag (remove dependencies and exit with verbose output)
	if cleanFlag {
		// Clean mode always has debug-level output
		platform.SetDebugMode(true)

		ui.PrintWarning("Clean mode: Removing existing dependencies...")
		installer, err := platform.GetInstaller()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Platform detection failed: %v", err))
			os.Exit(1)
		}

		if err := installer.CleanDependencies(); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to clean dependencies: %v", err))
			os.Exit(1)
		}
		ui.PrintSuccess("Dependencies cleaned successfully")
		fmt.Println()
		ui.PrintInfo("Clean complete. Run without -clean flag to install.")
		os.Exit(0)
	}

	// Start timer for the installation
	startTime := time.Now()

	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Installation start time: %s", startTime.Format(time.RFC3339)))
	}

	// Track current step number dynamically
	currentStep := 0
	
	// Detect platform silently (already detected by install script)
	installer, err := platform.GetInstaller()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Platform detection failed: %v", err))
		os.Exit(1)
	}

	// Step 1: Check prerequisites
	currentStep++
	stepStart := time.Now()
	ui.PrintStep("Checking prerequisites", currentStep, totalSteps)
	missing, err := installer.CheckPrerequisites()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Prerequisites check failed: %v", err))
		os.Exit(1)
	}

	if len(missing) > 0 {
		ui.PrintWarning("Missing dependencies detected:")
		for _, dep := range missing {
			fmt.Printf("  • %s: %s\n", dep.Name, dep.Status)
		}
		fmt.Println()

		if !ui.PromptYesNo("Would you like to install missing dependencies?", true) {
			ui.PrintError("Cannot proceed without dependencies")
			os.Exit(1)
		}
	} else {
		ui.PrintSuccess("All prerequisites satisfied")
	}
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
	}

	// Step 3: Install dependencies (only if needed)
	if len(missing) > 0 {
		currentStep++
		stepStart = time.Now()
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
				os.Exit(1)
			}
		}

		// Install remaining dependencies
		if err := installer.InstallDependencies(); err != nil {
			ui.PrintError(fmt.Sprintf("Dependency installation failed: %v", err))
			os.Exit(1)
		}

		ui.PrintSuccess("All dependencies installed")
		if debugFlag {
			ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
		}
	}

	// Step 4: Check J-Link probe
	currentStep++
	stepStart = time.Now()
	ui.PrintStep("Checking for J-Link probe", currentStep, totalSteps)

	// Retry loop for probe detection
	probeDetected := false
	for !probeDetected {
		if installer.CheckJLinkProbe() {
			ui.PrintSuccess("J-Link probe detected")
			probeDetected = true
		} else {
			ui.PrintWarning("No J-Link probe detected")
			ui.PrintInfo("Please ensure:")
			ui.PrintInfo("  • Developer board is connected via USB")
			ui.PrintInfo("  • Using a data cable (not charge-only)")
			ui.PrintInfo("  • Board is powered on")
			fmt.Println()

			// Ask what to do
			options := []string{
				"Retry - Check for probe again",
				"Continue anyway - Proceed without probe",
				"Exit - Cancel installation",
			}
			choice := ui.PromptChoice("What would you like to do?", options)

			switch choice {
			case 0: // Retry
				fmt.Println()
				ui.PrintInfo("Checking again...")
				continue
			case 1: // Continue anyway
				ui.PrintWarning("Continuing without probe detection")
				probeDetected = true
			case 2: // Exit
				os.Exit(0)
			}
		}
	}

	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
	}

	// Step 5: Get credentials
	currentStep++
	stepStart = time.Now()
	ui.PrintStep("Configuring credentials", currentStep, totalSteps)
	cfg, err := config.PromptForConfig()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Configuration failed: %v", err))
		os.Exit(1)
	}
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
	}

	// Step 6: Select board
	currentStep++
	stepStart = time.Now()
	ui.PrintStep("Selecting developer board", currentStep, totalSteps)
	boardOptions := make([]string, len(boards.AvailableBoards))
	for i, board := range boards.AvailableBoards {
		boardOptions[i] = fmt.Sprintf("%s - %s (%s)", board.Name, board.Description, board.Vendor)
	}

	selectedIndex := ui.PromptChoice("Available developer boards:", boardOptions)
	selectedBoard := boards.AvailableBoards[selectedIndex]
	cfg.Board = selectedBoard.ID

	ui.PrintSuccess(fmt.Sprintf("Selected: %s", selectedBoard.Name))
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		ui.PrintError(fmt.Sprintf("Invalid configuration: %v", err))
		os.Exit(1)
	}

	// Confirm before flashing
	fmt.Println()
	ui.PrintSuccess("All prerequisites installed!")
	if !ui.PromptYesNo(fmt.Sprintf("Would you like to flash and add your %s to your Hubble Network organization?", selectedBoard.Name), true) {
		ui.PrintWarning("Flashing skipped. You can flash later using:")
		fmt.Printf("  uv tool run --from pyhubbledemo hubbledemo flash %s -o %s -t <your_token>\n", cfg.Board, cfg.OrgID)
		os.Exit(0)
	}

	// Step 7: Flash the board
	currentStep++
	stepStart = time.Now()
	ui.PrintStep("Flashing board", currentStep, totalSteps)
	if err := installer.FlashBoard(cfg.OrgID, cfg.APIToken, cfg.Board); err != nil {
		ui.PrintError(fmt.Sprintf("Board flashing failed: %v", err))
		os.Exit(1)
	}
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step %d took: %v", currentStep, time.Since(stepStart)))
	}

	// Verify installation
	fmt.Println()
	ui.PrintInfo("Verifying installation...")
	if err := installer.Verify(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Verification warning: %v", err))
	}

	// Calculate total time
	duration := time.Since(startTime)

	// Print completion banner
	ui.PrintCompletionBanner(duration, cfg.OrgID, cfg.APIToken, debugFlag)

	// Success!
	os.Exit(0)
}
