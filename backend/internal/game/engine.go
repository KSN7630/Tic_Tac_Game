package game

import (
	"fmt"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
)

// GameEngine contains pure game logic and uses models and scoring.
type GameEngine struct {
	Scorer *scoring.ScoringEngine
}

func NewEngine(s *scoring.ScoringEngine) *GameEngine {
	return &GameEngine{
		Scorer: s,
	}
}

// ApplyMove applies a move to the board, updates state, and returns the score delta.
func (ge *GameEngine) ApplyMove(b *models.BoardState, playerID string, index int, elapsed time.Duration, usedGrace bool) (int, error) {
	if b == nil {
		return 0, appErr.New(appErr.ErrNilBoard, "the board is not initialized")
	}
	if b.Finished {
		return 0, appErr.New(appErr.ErrGameFinished, "the game has already finished")
	}
	if index < 0 || index > 8 {
		return 0, appErr.New(
			appErr.ErrIndexOutOfRange,
			fmt.Sprintf("move index %d is outside the valid range (0-8)", index),
		)
	}
	if b.Cells[index] != "" {
		return 0, appErr.New(
			appErr.ErrCellOccupied,
			fmt.Sprintf("cell %d is already occupied", index),
		)
	}
	if b.Turn != playerID {
		return 0, appErr.New(appErr.ErrNotPlayersTurn, "It is not your turn to move")
	}

	// Determine mark for player
	mark := ge.playerMark(b, playerID)
	if mark == "" {
		return 0, appErr.New(
			appErr.ErrUnknownPlayerMark,
			"invalid or unknown player identifier for marking",
		)
	}

	// Apply move
	b.Cells[index] = mark
	b.Moves = append(b.Moves, models.Move{PlayerID: playerID, Index: index, At: time.Now()})

	// Scoring for this move
	timedOut := elapsed > ge.Scorer.MaxMoveTime
	scoreDelta := ge.Scorer.ScoreMove(elapsed, timedOut, usedGrace)

	// Check for winner (returns "X" or "O")
	if winnerMark := ge.checkWinner(b); winnerMark != "" {
		b.Finished = true
		// Map mark -> actual player ID
		if winnerMark == "X" {
			b.Winner = b.PlayerX
		} else {
			b.Winner = b.PlayerO
		}
		scoreDelta += ge.Scorer.ScoreWin()
	}

	// Check draw
	if !b.Finished && ge.boardFull(b) {
		b.Finished = true
		b.Winner = "" // draw
	}

	// Switch turn if game continues
	if !b.Finished {
		if b.Turn == b.PlayerX {
			b.Turn = b.PlayerO
		} else {
			b.Turn = b.PlayerX
		}
	}

	return scoreDelta, nil
}

// playerMark returns "X" or "O" for the current player.
func (ge *GameEngine) playerMark(b *models.BoardState, playerID string) string {
	if playerID == b.PlayerX {
		return "X"
	}
	if playerID == b.PlayerO {
		return "O"
	}
	return ""
}

// boardFull checks if all cells are filled.
func (ge *GameEngine) boardFull(b *models.BoardState) bool {
	for _, c := range b.Cells {
		if c == "" {
			return false
		}
	}
	return true
}

// checkWinner returns "X" or "O" if there's a winner, otherwise "".
func (ge *GameEngine) checkWinner(b *models.BoardState) string {
	lines := [8][3]int{
		{0, 1, 2},
		{3, 4, 5},
		{6, 7, 8},
		{0, 3, 6},
		{1, 4, 7},
		{2, 5, 8},
		{0, 4, 8},
		{2, 4, 6},
	}
	for _, l := range lines {
		a, bb, c := l[0], l[1], l[2]
		if b.Cells[a] != "" && b.Cells[a] == b.Cells[bb] && b.Cells[a] == b.Cells[c] {
			return b.Cells[a]
		}
	}
	return ""
}
