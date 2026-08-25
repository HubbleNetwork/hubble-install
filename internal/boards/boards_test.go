package boards

import "testing"

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
