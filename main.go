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

	// Step 1: Detect platform
	stepStart := time.Now()
	ui.PrintStep("Detecting platform", 1, totalSteps)
	installer, err := platform.GetInstaller()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Platform detection failed: %v", err))
		os.Exit(1)
	}
	ui.PrintSuccess(fmt.Sprintf("Platform detected: %s", installer.Name()))
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step 1 took: %v", time.Since(stepStart)))
	}

	// Step 2: Get credentials
	stepStart = time.Now()
	ui.PrintStep("Configuring credentials", 2, totalSteps)
	cfg, err := config.PromptForConfig()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Configuration failed: %v", err))
		os.Exit(1)
	}
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step 2 took: %v", time.Since(stepStart)))
	}

	// Step 3: Select board
	stepStart = time.Now()
	ui.PrintStep("Selecting developer board", 3, totalSteps)
	boardOptions := make([]string, len(boards.AvailableBoards))
	for i, board := range boards.AvailableBoards {
		boardOptions[i] = fmt.Sprintf("%s - %s (%s)", board.Name, board.Description, board.Vendor)
	}

	selectedIndex := ui.PromptChoice("Available developer boards:", boardOptions)
	selectedBoard := boards.AvailableBoards[selectedIndex]
	cfg.Board = selectedBoard.ID

	ui.PrintSuccess(fmt.Sprintf("Selected: %s", selectedBoard.Name))
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step 3 took: %v", time.Since(stepStart)))
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		ui.PrintError(fmt.Sprintf("Invalid configuration: %v", err))
		os.Exit(1)
	}

	// Step 4: Check prerequisites
	stepStart = time.Now()
	ui.PrintStep("Checking prerequisites", 4, totalSteps)
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
		ui.PrintDebug(fmt.Sprintf("Step 4 took: %v", time.Since(stepStart)))
	}

	// Step 5: Install dependencies
	if len(missing) > 0 {
		stepStart = time.Now()
		ui.PrintStep("Installing dependencies", 5, totalSteps)

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
			ui.PrintDebug(fmt.Sprintf("Step 5 took: %v", time.Since(stepStart)))
		}
	}

	// Confirm before flashing
	fmt.Println()
	ui.PrintSuccess("All prerequisites installed!")
	if !ui.PromptYesNo(fmt.Sprintf("Would you like to flash and add your %s to your Hubble Network organization?", selectedBoard.Name), true) {
		ui.PrintWarning("Flashing skipped. You can flash later using:")
		fmt.Printf("  uvx --from pyhubbledemo hubbledemo flash %s -o <org_id> -t <api_token>\n", cfg.Board)
		os.Exit(0)
	}

	// Step 6: Flash the board
	stepStart = time.Now()
	ui.PrintStep("Flashing board", 6, totalSteps)
	if err := installer.FlashBoard(cfg.OrgID, cfg.APIToken, cfg.Board); err != nil {
		ui.PrintError(fmt.Sprintf("Board flashing failed: %v", err))
		os.Exit(1)
	}
	if debugFlag {
		ui.PrintDebug(fmt.Sprintf("Step 6 took: %v", time.Since(stepStart)))
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
	ui.PrintCompletionBanner(duration)

	// Success!
	os.Exit(0)
}
