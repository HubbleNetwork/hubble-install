package boards

import "fmt"

// Board represents a developer board that can be flashed
type Board struct {
	ID          string
	Name        string
	Description string
	Vendor      string
}

// Available boards for flashing
var AvailableBoards = []Board{
	{
		ID:          "nrf21540dk",
		Name:        "nRF21540 DK",
		Description: "Nordic Semiconductor nRF21540 Development Kit",
		Vendor:      "Nordic",
	},
	{
		ID:          "nrf52840dk",
		Name:        "nRF52840 DK",
		Description: "Nordic Semiconductor nRF52840 Development Kit",
		Vendor:      "Nordic",
	},
	{
		ID:          "ticc2340r5",
		Name:        "TI CC2340R5",
		Description: "Texas Instruments CC2340R5 Development Kit",
		Vendor:      "Texas Instruments",
	},
	// {
	// 	ID:          "nrf52dk",
	// 	Name:        "nRF52 DK",
	// 	Description: "Nordic Semiconductor nRF52 Development Kit",
	// 	Vendor:      "Nordic",
	// },
	// {
	// 	ID:          "xg22_ek4108a",
	// 	Name:        "xG22 EK4108A",
	// 	Description: "Silicon Labs xG22 Explorer Kit",
	// 	Vendor:      "Silicon Labs",
	// },
	// {
	// 	ID:          "xg24_ek2703a",
	// 	Name:        "xG24 EK2703A",
	// 	Description: "Silicon Labs xG24 Explorer Kit",
	// 	Vendor:      "Silicon Labs",
	// },
}

// GetBoard returns a board by its ID
func GetBoard(id string) (*Board, error) {
	for _, board := range AvailableBoards {
		if board.ID == id {
			return &board, nil
		}
	}
	return nil, fmt.Errorf("board not found: %s", id)
}

// FormatBoardList returns a formatted string of all available boards
func FormatBoardList() string {
	result := ""
	for i, board := range AvailableBoards {
		result += fmt.Sprintf("%d. %s - %s\n", i+1, board.Name, board.Description)
	}
	return result
}
