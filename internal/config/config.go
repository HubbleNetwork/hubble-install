package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// Config holds the Hubble configuration
type Config struct {
	OrgID    string
	APIToken string
	Board    string
}

// PromptForConfig prompts the user for all required configuration
// Returns the config and a boolean indicating if credentials were pre-configured
func PromptForConfig() (*Config, bool, error) {
	config := &Config{}
	preConfigured := false

	// Check for base64 encoded credentials first (passed from install.sh)
	if encodedCreds := os.Getenv("HUBBLE_CREDENTIALS"); encodedCreds != "" {
		decoded, err := base64.StdEncoding.DecodeString(encodedCreds)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				config.OrgID = strings.TrimSpace(parts[0])
				config.APIToken = strings.TrimSpace(parts[1])
				if config.OrgID != "" && config.APIToken != "" {
					preConfigured = true
					return config, preConfigured, nil
				}
			}
		}
	}

	// Check environment variables
	envOrgID := os.Getenv("HUBBLE_ORG_ID")
	envAPIToken := os.Getenv("HUBBLE_API_TOKEN")

	// If both are present, use them
	if envOrgID != "" && envAPIToken != "" {
		config.OrgID = envOrgID
		config.APIToken = envAPIToken
		preConfigured = true
		ui.PrintSuccess("Credentials found in environment")
		return config, preConfigured, nil
	}

	// Print info about where to find credentials
	ui.PrintInfo("Get your credentials at: https://dash.hubble.com/developer/api-tokens")
	fmt.Println()

	// Prompt for Org ID (if not in environment)
	if envOrgID != "" {
		config.OrgID = envOrgID
		ui.PrintSuccess(fmt.Sprintf("Using Org ID from environment: %s", envOrgID))
	} else {
		for {
			orgID := ui.PromptInput("Enter your Hubble Org ID")
			orgID = strings.TrimSpace(orgID)
			if orgID != "" {
				config.OrgID = orgID
				break
			}
			ui.PrintWarning("Org ID cannot be empty")
		}
	}

	// Prompt for API Token (if not in environment)
	if envAPIToken != "" {
		config.APIToken = envAPIToken
		ui.PrintSuccess("Using API Token from environment")
	} else {
		for {
			apiToken := ui.PromptPassword("Enter your Hubble API Token (hidden)")
			apiToken = strings.TrimSpace(apiToken)
			if apiToken != "" {
				config.APIToken = apiToken
				break
			}
			ui.PrintWarning("API Token cannot be empty")
		}
	}

	ui.PrintSuccess("Credentials configured")

	return config, preConfigured, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OrgID == "" {
		return fmt.Errorf("org ID is required")
	}
	if c.APIToken == "" {
		return fmt.Errorf("API token is required")
	}
	if c.Board == "" {
		return fmt.Errorf("board selection is required")
	}
	return nil
}
