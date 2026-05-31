package game

import (
	"testing"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
)

func newBoard() *models.BoardState {
	b := models.NewEmptyBoard("g1", "px")
	b.PlayerX = "px"
	b.PlayerO = "po"
	b.Turn = "px"
	return b
}

func engine() *GameEngine {
	return NewEngine(scoring.New())
}

func codeOf(t *testing.T, err error) appErr.ErrorCode {
	t.Helper()
	ae, ok := appErr.As(err)
	if !ok {
		t.Fatalf("expected AppError, got %v", err)
	}
	return ae.Code
}

func TestApplyMove_RejectsWrongTurn(t *testing.T) {
	ge := engine()
	b := newBoard()
	// po tries to move first, but it's px's turn
	_, err := ge.ApplyMove(b, "po", 0, time.Second, false)
	if got := codeOf(t, err); got != appErr.ErrNotPlayersTurn {
		t.Fatalf("want %s, got %s", appErr.ErrNotPlayersTurn, got)
	}
}

func TestApplyMove_RejectsOccupiedCell(t *testing.T) {
	ge := engine()
	b := newBoard()
	if _, err := ge.ApplyMove(b, "px", 0, time.Second, false); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// now po's turn, plays same cell
	_, err := ge.ApplyMove(b, "po", 0, time.Second, false)
	if got := codeOf(t, err); got != appErr.ErrCellOccupied {
		t.Fatalf("want %s, got %s", appErr.ErrCellOccupied, got)
	}
}

func TestApplyMove_RejectsOutOfRange(t *testing.T) {
	ge := engine()
	b := newBoard()
	_, err := ge.ApplyMove(b, "px", 9, time.Second, false)
	if got := codeOf(t, err); got != appErr.ErrIndexOutOfRange {
		t.Fatalf("want %s, got %s", appErr.ErrIndexOutOfRange, got)
	}
}

func TestApplyMove_TurnSwitches(t *testing.T) {
	ge := engine()
	b := newBoard()
	if _, err := ge.ApplyMove(b, "px", 0, time.Second, false); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if b.Turn != "po" {
		t.Fatalf("turn should pass to po, got %s", b.Turn)
	}
}

func TestApplyMove_DetectsWin(t *testing.T) {
	ge := engine()
	b := newBoard()
	// px: 0,1,2 (top row). po: 3,4 between.
	seq := []struct {
		player string
		idx    int
	}{
		{"px", 0},
		{"po", 3},
		{"px", 1},
		{"po", 4},
		{"px", 2}, // px wins top row
	}
	for _, m := range seq {
		if _, err := ge.ApplyMove(b, m.player, m.idx, time.Second, false); err != nil {
			t.Fatalf("move %+v failed: %v", m, err)
		}
	}
	if !b.Finished {
		t.Fatal("game should be finished")
	}
	if b.Winner != "px" {
		t.Fatalf("winner should be px, got %q", b.Winner)
	}
}

func TestApplyMove_DetectsDraw(t *testing.T) {
	ge := engine()
	b := newBoard()
	// A full board with no winner:
	// X O X
	// X O O
	// O X X
	order := []struct {
		player string
		idx    int
	}{
		{"px", 0}, {"po", 1}, {"px", 2},
		{"po", 4}, {"px", 3}, {"po", 5},
		{"px", 7}, {"po", 6}, {"px", 8},
	}
	for _, m := range order {
		if _, err := ge.ApplyMove(b, m.player, m.idx, time.Second, false); err != nil {
			t.Fatalf("move %+v failed: %v", m, err)
		}
	}
	if !b.Finished {
		t.Fatal("board full -> game should be finished")
	}
	if b.Winner != "" {
		t.Fatalf("draw expected, got winner %q", b.Winner)
	}
}

func TestApplyMove_RejectsAfterFinished(t *testing.T) {
	ge := engine()
	b := newBoard()
	b.Finished = true
	_, err := ge.ApplyMove(b, "px", 0, time.Second, false)
	if got := codeOf(t, err); got != appErr.ErrGameFinished {
		t.Fatalf("want %s, got %s", appErr.ErrGameFinished, got)
	}
}
