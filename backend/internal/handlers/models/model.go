package models

import "errors"

// Message is the schema for messages exchanged via websocket.

type Message struct {
	Type      string `json:"type"`      // "group_chat" or "match_chat"
	GameID    string `json:"game_id"`   // which game/room (for match_chat)
	Sender    string `json:"sender"`    // player id
	Recipient string `json:"recipient"` // optional for private msg
	Content   string `json:"content"`   // text
	Timestamp int64  `json:"timestamp"` // unix timestamp (filled server-side)
}

type Move struct {
	GameID string `json:"game_id"`
	Player string `json:"player"`
	Cell   int    `json:"cell"` // 0–8 for 3x3 board
	Symbol string `json:"symbol"`
}

//----------------------------------------------------------//
type BoardState struct {
	Cells   [9]string `json:"cells"`
	Winner  string    `json:"winner"`
	Turn    string    `json:"turn"`
	Status  string    `json:"status"` // waiting, in_progress, finished
	PlayerX string    `json:"player_x"`
	PlayerO string    `json:"player_o"`
}

// MakeMove validates and applies a move.
func (b *BoardState) MakeMove(move Move) error {
	if b.Status != "in_progress" {
		return errors.New("game already finished")
	}
	if move.Player != b.Turn {
		return errors.New("not your turn")
	}
	if move.Cell < 0 || move.Cell > 8 {
		return errors.New("invalid cell")
	}
	if b.Cells[move.Cell] != "" {
		return errors.New("cell already occupied")
	}

	b.Cells[move.Cell] = move.Symbol

	// Check win/draw
	if checkWinner(b.Cells, move.Symbol) {
		b.Status = "finished"
		b.Winner = move.Player
	} else if checkDraw(b.Cells) {
		b.Status = "finished"
		b.Winner = "draw"
	} else {
		// Switch turn
		if b.Turn == b.PlayerX {
			b.Turn = b.PlayerO
		} else {
			b.Turn = b.PlayerX
		}
	}
	return nil
}

// checkWinner checks if a symbol has won.
func checkWinner(cells [9]string, sym string) bool {
	winPatterns := [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
		{0, 4, 8}, {2, 4, 6}, // diagonals
	}
	for _, p := range winPatterns {
		if cells[p[0]] == sym && cells[p[1]] == sym && cells[p[2]] == sym {
			return true
		}
	}
	return false
}

func checkDraw(cells [9]string) bool {
	for _, c := range cells {
		if c == "" {
			return false
		}
	}
	return true
}

// Room structure (partial mirror from lobby)
type RoomWithTurn struct {
	RoomID    string `json:"room_id"`
	PlayerX   string `json:"player_x"`
	PlayerO   string `json:"player_o"`
	Status    string `json:"status"`
	GameType  string `json:"game_type"`
	Created   int64  `json:"created"`
	Turn      string `json:"turn,omitempty"`
	FirstTurn string `json:"first_turn,omitempty"`
}
