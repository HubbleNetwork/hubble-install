package boards

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/HubbleNetwork/hubble-install/internal/ui"
)

// Flash methods. These are md.json's own "method" values, consumed verbatim so
// the installer and the manifest share one vocabulary rather than maintaining a
// translated copy that can drift.
const (
	FlashMethodJLink = "jlink-flash"  // Direct flash via SEGGER J-Link
	FlashMethodHex   = "generate-hex" // Generate a hex file to flash manually
)

// mdJSONURL is the source of truth for supported boards. It is maintained in the
// hubble-tldm repo and also fetched by pyhubbledemo at flash time, so the
// installer reads the same list rather than duplicating it.
const mdJSONURL = "https://raw.githubusercontent.com/HubbleNetwork/hubble-tldm/master/merge/md.json"

// Board represents a developer board that can be flashed
type Board struct {
	ID          string
	Name        string
	FlashMethod string // md.json "method": FlashMethodJLink or FlashMethodHex
	Workspace   string // md.json "workspace" (e.g. zephyr, sat-zephyr)
}

// mdEntry mirrors the per-board shape of md.json. Only the fields the installer
// needs are decoded; the rest (jlink_device, board_target, artifact, ...) are
// consumed by pyhubbledemo.
type mdEntry struct {
	Name      string `json:"name"`
	Method    string `json:"method"` // "jlink-flash" or "generate-hex"
	Workspace string `json:"workspace"`
}

// RequiresJLink returns true if this board requires SEGGER J-Link
func (b *Board) RequiresJLink() bool {
	return b.FlashMethod == FlashMethodJLink
}

// IsSatellite reports whether this board flashes a satellite (satnet) image.
// Satellite workspaces in md.json are prefixed with "sat-" (e.g. sat-zephyr);
// board IDs for those images also conventionally end in "_sat".
func (b *Board) IsSatellite() bool {
	if strings.HasPrefix(b.Workspace, "sat-") {
		return true
	}
	return strings.HasSuffix(b.ID, "_sat")
}

// GetDependencies returns the list of dependencies required for this board
func (b *Board) GetDependencies() []string {
	if b.RequiresJLink() {
		// Nordic DKs use a J-Link probe (often on-board). We need:
		// - uv: to run our Python flashing tool (pyhubbledemo)
		// - segger-jlink: provides J-Link drivers/DLLs that pylink-square uses
		return []string{"uv", "segger-jlink"}
	}
	return []string{"uv"}
}

// FetchBoards downloads md.json from hubble-tldm and converts it to the
// installer's board model. The tool requires network connectivity anyway
// (pyhubbledemo fetches the same file and downloads firmware), so any failure
// here is surfaced to the caller rather than masked with a stale fallback.
func FetchBoards() ([]Board, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(mdJSONURL)
	if err != nil {
		return nil, fmt.Errorf("could not fetch the board list from %s: %w", mdJSONURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not fetch the board list from %s: HTTP %d", mdJSONURL, resp.StatusCode)
	}

	var raw map[string]mdEntry
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("could not parse the board list from %s: %w", mdJSONURL, err)
	}

	available := make([]Board, 0, len(raw))
	for id, entry := range raw {
		if !knownMethod(entry.Method) {
			// Skip boards whose flash method the installer doesn't understand
			// so a new upstream method can't break board selection entirely.
			// Warn so a board going missing isn't silent (likely a stale installer).
			ui.PrintWarning(fmt.Sprintf("Skipping board %q: unsupported flash method %q (this installer may be out of date)", id, entry.Method))
			continue
		}
		available = append(available, Board{
			ID:          id,
			Name:        entry.Name,
			FlashMethod: entry.Method,
			Workspace:   entry.Workspace,
		})
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("no supported boards found in the board list at %s", mdJSONURL)
	}

	// md.json is a JSON object, so iteration order is random. Sort by name for a
	// stable menu; because every name is vendor-prefixed, this groups vendors
	// contiguously without a separate vendor field to keep in sync.
	sort.Slice(available, func(i, j int) bool {
		return available[i].Name < available[j].Name
	})

	return available, nil
}

// knownMethod reports whether an md.json "method" is one the installer can flash.
func knownMethod(method string) bool {
	return method == FlashMethodJLink || method == FlashMethodHex
}

// GetBoard returns the board with the given ID from the provided list.
func GetBoard(available []Board, id string) (*Board, error) {
	for i := range available {
		if available[i].ID == id {
			return &available[i], nil
		}
	}
	return nil, fmt.Errorf("board not found: %s", id)
}
