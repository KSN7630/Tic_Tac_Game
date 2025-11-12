package models

import "time"

// Player represents a connected player.
type Player struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Move represents a single move on the Tic-Tac-Toe board.

type Move struct {
	PlayerID string    `json:"player_id"`
	Index    int       `json:"index"`
	At       time.Time `json:"at"`
}

// BoardState is the authoritative board model for a room/game.
type BoardState struct {
	GameID   string    `json:"game_id"`
	Cells    [9]string `json:"cells"`
	Turn     string    `json:"turn"`     // whose turn it is
	PlayerX  string    `json:"player_x"` // Player assigned "X"
	PlayerO  string    `json:"player_o"` // Player assigned "O"
	Winner   string    `json:"winner"`   // PlayerID of the winner
	Finished bool      `json:"finished"`
	Moves    []Move    `json:"moves"` // record of all moves
	Created  time.Time `json:"created"`
}

// NewEmptyBoard returns a fresh board with zeroed cells.
func NewEmptyBoard(gameID, firstTurn string) *BoardState {
	return &BoardState{
		GameID:   gameID,
		Cells:    [9]string{},
		PlayerX:  firstTurn,
		PlayerO:  "",        // will be set when second player joins
		Turn:     firstTurn, // X always starts first
		Finished: false,
		Moves:    []Move{},
		Created:  time.Now(),
	}
}
