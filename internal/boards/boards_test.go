package boards

import (
	"slices"
	"testing"
)

func TestBoardIsSatellite(t *testing.T) {
	tests := []struct {
		name  string
		board Board
		want  bool
	}{
		{
			name:  "sat-zephyr workspace",
			board: Board{ID: "nrf21540dk_sat", Workspace: "sat-zephyr"},
			want:  true,
		},
		{
			name:  "sat-zephyr xg24 board",
			board: Board{ID: "xg24_rb4187c_sat", Workspace: "sat-zephyr"},
			want:  true,
		},
		{
			name:  "terrestrial zephyr workspace",
			board: Board{ID: "nrf21540dk", Workspace: "zephyr"},
			want:  false,
		},
		{
			name:  "ti terrestrial workspace",
			board: Board{ID: "lp_em_cc2340r5", Workspace: "ti-zephyr"},
			want:  false,
		},
		{
			name:  "id suffix fallback when workspace missing",
			board: Board{ID: "nrf21540dk_sat", Workspace: ""},
			want:  true,
		},
		{
			name:  "no satellite signals",
			board: Board{ID: "nrf52dk", Workspace: ""},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.board.IsSatellite(); got != tt.want {
				t.Fatalf("IsSatellite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoardFlashMethod(t *testing.T) {
	tests := []struct {
		method          string
		known           bool
		requiresJLink   bool
		flashesDirectly bool
	}{
		{method: FlashMethodJLink, known: true, requiresJLink: true, flashesDirectly: true},
		{method: FlashMethodESPTool, known: true, requiresJLink: false, flashesDirectly: true},
		{method: FlashMethodHex, known: true, requiresJLink: false, flashesDirectly: false},
		{method: "some-future-method", known: false, requiresJLink: false, flashesDirectly: false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			b := Board{FlashMethod: tt.method}
			if got := knownMethod(tt.method); got != tt.known {
				t.Errorf("knownMethod() = %v, want %v", got, tt.known)
			}
			if got := b.RequiresJLink(); got != tt.requiresJLink {
				t.Errorf("RequiresJLink() = %v, want %v", got, tt.requiresJLink)
			}
			if got := b.FlashesDirectly(); got != tt.flashesDirectly {
				t.Errorf("FlashesDirectly() = %v, want %v", got, tt.flashesDirectly)
			}
		})
	}
}

func TestBoardGetDependencies(t *testing.T) {
	tests := []struct {
		method string
		want   []string
	}{
		{method: FlashMethodJLink, want: []string{"uv", "segger-jlink"}},
		{method: FlashMethodESPTool, want: []string{"uv", "serial-port-access"}},
		{method: FlashMethodHex, want: []string{"uv"}},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			b := Board{FlashMethod: tt.method}
			if got := b.GetDependencies(); !slices.Equal(got, tt.want) {
				t.Fatalf("GetDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}
