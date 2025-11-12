package storage

import (
	"context"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
)

// Storage defines unified data access interface for both Redis and Postgres.
type Storage interface {
	// --- Redis-style ephemeral operations ---
	SaveBoard(ctx context.Context, gameID string, b *models.BoardState) error
	LoadBoard(ctx context.Context, gameID string) (*models.BoardState, error)
	SaveChatMessage(ctx context.Context, key string, msg interface{}, ttl time.Duration) error
	LoadChatMessages(ctx context.Context, key string) ([]string, error)
	SaveReplayMove(ctx context.Context, gameID string, move interface{}, ttl time.Duration) error
	SaveUser(ctx context.Context, player models.Player, ttl time.Duration) error
	SaveRoom(ctx context.Context, roomID string, data interface{}, ttl time.Duration) error
	// --- Postgres-style persistent operations ---
	SaveGameSummary(ctx context.Context, gameID, winnerID string) error
	SavePlayerScore(ctx context.Context, gameID, playerID string, score int) error
	GetLeaderboard(ctx context.Context, gameType string) ([]LeaderboardEntry, error)

	SaveGroupChat(ctx context.Context, payload map[string]any) error
	SaveMatchChat(ctx context.Context, roomID string, payload map[string]any) error

	Close() error
}

// LeaderboardEntry represents a single record in leaderboard.
type LeaderboardEntry struct {
	PlayerID string
	Score    int
	GameType string
}
