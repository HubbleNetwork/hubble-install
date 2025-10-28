package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

var (
	cyan   = color.New(color.FgCyan, color.Bold)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	blue   = color.New(color.FgBlue, color.Bold)
	bold   = color.New(color.Bold)
)

// PrintBanner prints the welcome banner
func PrintBanner() {
	cyan.Println(`
╔═══════════════════════════════════════════════════════════╗
║      Welcome to Hubble Network! Let's get you setup.      ║
╚═══════════════════════════════════════════════════════════╝
`)
}

// PrintStep prints a step indicator
func PrintStep(step string, current, total int) {
	fmt.Println()
	blue.Printf("[%d/%d] %s\n", current, total, step)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	green.Printf("✓ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	red.Printf("✗ %s\n", message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	yellow.Printf("⚠ %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	cyan.Printf("ℹ %s\n", message)
}

// PrintDebug prints a debug message (gray color)
func PrintDebug(message string) {
	gray := color.New(color.FgHiBlack)
	gray.Printf("🔍 [DEBUG] %s\n", message)
}

// Global reader for interactive input
var stdinReader *bufio.Reader

func init() {
	// Try to open /dev/tty for interactive input (works when piped from curl)
	tty, err := os.Open("/dev/tty")
	if err == nil {
		stdinReader = bufio.NewReader(tty)
	} else {
		// Fallback to stdin if /dev/tty is not available
		stdinReader = bufio.NewReader(os.Stdin)
	}
}

// PromptInput prompts the user for input
func PromptInput(prompt string) string {
	cyan.Printf("? %s: ", prompt)
	input, err := stdinReader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, something is seriously wrong
		PrintError(fmt.Sprintf("Failed to read input: %v", err))
		os.Exit(1)
	}
	return strings.TrimSpace(input)
}

// PromptPassword prompts the user for a password (masked input)
func PromptPassword(prompt string) string {
	cyan.Printf("? %s: ", prompt)

	// Try to open /dev/tty for password input
	tty, err := os.Open("/dev/tty")
	if err != nil {
		// Fallback to regular input if /dev/tty not available
		PrintWarning("Cannot access terminal, reading password as plain text")
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			PrintError(fmt.Sprintf("Failed to read password: %v", err))
			os.Exit(1)
		}
		return strings.TrimSpace(input)
	}
	defer tty.Close()

	fd := int(tty.Fd())

	// Check if it's actually a terminal
	if !term.IsTerminal(fd) {
		// Not a terminal, fall back to regular input
		PrintWarning("Not a terminal, reading password as plain text")
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			PrintError(fmt.Sprintf("Failed to read password: %v", err))
			os.Exit(1)
		}
		return strings.TrimSpace(input)
	}

	// Terminal mode - read password with masking from /dev/tty
	bytePassword, err := term.ReadPassword(fd)
	fmt.Println() // Add newline after password input

	if err != nil {
		PrintError(fmt.Sprintf("Failed to read password: %v", err))
		os.Exit(1)
	}

	result := string(bytePassword)
	if result == "" {
		PrintDebug("Empty password received from term.ReadPassword")
	}

	return result
}

// PromptYesNo prompts the user for a yes/no answer
func PromptYesNo(question string, defaultYes bool) bool {
	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}

	for {
		cyan.Printf("? %s (%s): ", question, defaultStr)
		response, err := stdinReader.ReadString('\n')
		if err != nil {
			PrintError(fmt.Sprintf("Failed to read input: %v", err))
			os.Exit(1)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" {
			return defaultYes
		}
		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}
		PrintWarning("Please answer 'y' or 'n'")
	}
}

// PromptChoice prompts the user to select from a list of options
func PromptChoice(prompt string, options []string) int {
	fmt.Println()
	cyan.Println(prompt)
	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}

	for {
		cyan.Printf("? Select (1-%d): ", len(options))
		response, err := stdinReader.ReadString('\n')
		if err != nil {
			PrintError(fmt.Sprintf("Failed to read input: %v", err))
			os.Exit(1)
		}
		response = strings.TrimSpace(response)

		var choice int
		_, err = fmt.Sscanf(response, "%d", &choice)
		if err == nil && choice >= 1 && choice <= len(options) {
			return choice - 1
		}
		PrintWarning(fmt.Sprintf("Please enter a number between 1 and %d", len(options)))
	}
}

// PrintCompletionBanner prints the success completion banner
func PrintCompletionBanner(duration time.Duration) {
	green.Println(`
╔═══════════════════════════════════════════════════════════╗
║     ✓ Installation Complete!                              ║
╚═══════════════════════════════════════════════════════════╝
`)

	cyan.Printf("⏱️  Total time: %.1f seconds\n\n", duration.Seconds())

	cyan.Println("Next steps:")
	fmt.Println()
	fmt.Print("  1. Flash additional boards:\n     ")
	bold.Println("uvx --from pyhubbledemo hubbledemo flash <board>")
	fmt.Println()
	fmt.Print("  2. View available commands:\n     ")
	bold.Println("uvx --from pyhubbledemo hubbledemo --help")
	fmt.Println()
	fmt.Print("  3. Documentation:\n     ")
	bold.Println("https://docs.hubble.com")
	fmt.Println()

	yellow.Println("Need help? Visit https://hubble.com/support/")
}

// Spinner represents a loading spinner
type Spinner struct {
	message string
	stop    chan bool
	done    chan bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan bool),
		done:    make(chan bool),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Print("\r\033[K") // Clear line
				s.done <- true
				return
			default:
				cyan.Printf("\r%s %s", chars[i], s.message)
				i = (i + 1) % len(chars)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.stop <- true
	<-s.done
}
