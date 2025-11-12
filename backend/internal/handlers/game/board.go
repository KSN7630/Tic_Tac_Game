package game

import (
	"encoding/json"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// LoadBoard loads the board for a room from Redis.
func LoadBoard(gameID string) (*models.BoardState, error) {
	ctx := storage.Ctx
	data, err := storage.Rdb.Get(ctx, "board:"+gameID).Bytes()
	if err != nil {
		return nil, err
	}

	var b models.BoardState
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// SaveBoard saves the current board state back to Redis.
func SaveBoard(gameID string, b *models.BoardState) error {
	ctx := storage.Ctx
	data, _ := json.Marshal(b)
	return storage.Rdb.Set(ctx, "board:"+gameID, data, 30*time.Minute).Err()
}

// NewBoard initializes an empty 3x3 board.
func NewBoard(playerX, playerO, firstTurn string) *models.BoardState {
	return &models.BoardState{
		Cells:   [9]string{},
		Winner:  "",
		Status:  "in_progress",
		PlayerX: playerX,
		PlayerO: playerO,
		Turn:    firstTurn,
	}
}
