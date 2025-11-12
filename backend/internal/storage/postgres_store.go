package storage

import (
	"context"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/jackc/pgx/v5"
)

// PostgresStore handles persistent data (scores, summaries, leaderboard)
type PostgresStore struct {
	Conn *pgx.Conn
}

// NewPostgresStore connects to PostgreSQL
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{Conn: conn}, nil
}

// SaveGameSummary inserts record into games table
func (p *PostgresStore) SaveGameSummary(ctx context.Context, gameID, winnerID string) error {
	_, err := p.Conn.Exec(ctx,
		"INSERT INTO games (game_id, game_type, winner_id) VALUES ($1, $2, $3)",
		gameID, "tic-tac-toe", winnerID)
	return err
}

// SavePlayerScore inserts player score into scores table
func (p *PostgresStore) SavePlayerScore(ctx context.Context, gameID, playerID string, score int) error {
	_, err := p.Conn.Exec(ctx,
		"INSERT INTO scores (game_id, player_id, score) VALUES ($1, $2, $3)",
		gameID, playerID, score)
	return err
}

// GetLeaderboard queries top players by total score
func (p *PostgresStore) GetLeaderboard(ctx context.Context, gameType string) ([]LeaderboardEntry, error) {
	rows, err := p.Conn.Query(ctx,
		`SELECT s.player_id, SUM(s.score) AS total_score, g.game_type
         FROM scores s JOIN games g ON s.game_id = g.game_id
         WHERE g.game_type=$1 GROUP BY s.player_id, g.game_type
         ORDER BY total_score DESC LIMIT 10`, gameType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.PlayerID, &e.Score, &e.GameType); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// --- Redis-only methods ---
func (p *PostgresStore) SaveBoard(ctx context.Context, gameID string, b any) error { return nil }
func (p *PostgresStore) LoadBoard(ctx context.Context, gameID string) (any, error) { return nil, nil }
func (p *PostgresStore) SaveChatMessage(ctx context.Context, key string, msg interface{}, ttl time.Duration) error {
	return nil
}
func (p *PostgresStore) LoadChatMessages(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}
func (p *PostgresStore) SaveReplayMove(ctx context.Context, gameID string, move interface{}, ttl time.Duration) error {
	return nil
}

func (p *PostgresStore) SaveUser(ctx context.Context, player models.Player, ttl time.Duration) error {
	return nil
}

func (p *PostgresStore) SaveRoom(ctx context.Context, roomID string, data interface{}, ttl time.Duration) error {
	return nil
}

func (p *PostgresStore) Close() error {
	return p.Conn.Close(context.Background())
}
