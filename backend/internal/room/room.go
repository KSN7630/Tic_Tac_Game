package room

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/events"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// Room represents one running match.
type Room struct {
	ID          string
	Status      string // waiting, in_progress, completed
	Players     []models.Player
	board       *models.BoardState
	mu          sync.Mutex
	engine      *game.GameEngine
	scorer      *scoring.ScoringEngine
	store       storage.Storage
	broadcaster events.Broadcaster
	scores      map[string]int
}

// NewRoom creates a new game room.
func NewRoom(id string, store storage.Storage, bc events.Broadcaster, eng *game.GameEngine, sc *scoring.ScoringEngine) *Room {
	return &Room{
		ID:          id,
		Status:      "waiting",
		store:       store,
		broadcaster: bc,
		engine:      eng,
		scorer:      sc,
		scores:      make(map[string]int),
		board:       &models.BoardState{GameID: id, Cells: [9]string{}, Created: time.Now()},
	}
}

// AddPlayer adds a player to the room and updates Redis.
func (r *Room) AddPlayer(p models.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Players) >= 2 {
		return fmt.Errorf("room full")
	}
	for _, existing := range r.Players {
		if existing.ID == p.ID {
			return nil
		}
	}

	r.Players = append(r.Players, p)
	r.scores[p.ID] = 0

	// ✅ Save room info in Redis after player joins
	if r.store != nil {
		roomData := map[string]interface{}{
			"room_id": r.ID,
			"players": r.Players,
			"status":  r.Status,
		}
		_ = r.store.SaveRoom(context.Background(), r.ID, roomData, 30*time.Minute)
	}

	return nil
}

// StartGame triggers start event and initializes board in Redis.
func (r *Room) StartGame(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Players) < 2 {
		return fmt.Errorf("need 2 players to start")
	}

	r.Status = "in_progress"

	// 🎯 Toss logic — randomly decide who starts (fair chance)
	rand.Seed(time.Now().UnixNano())
	startIndex := rand.Intn(2) // 0 or 1

	firstTurn := r.Players[startIndex].ID
	secondPlayer := r.Players[1-startIndex].ID

	// ✅ Initialize board with PlayerX (first) and PlayerO (second)
	r.board = models.NewEmptyBoard(r.ID, firstTurn)
	r.board.PlayerX = firstTurn
	r.board.PlayerO = secondPlayer
	r.board.Turn = firstTurn // PlayerX starts first

	// Save initial board state in Redis
	if err := r.store.SaveBoard(ctx, r.ID, r.board); err != nil {
		return fmt.Errorf("save board: %w", err)
	}

	// ✅ Update room state in Redis
	if r.store != nil {
		roomData := map[string]interface{}{
			"room_id":  r.ID,
			"players":  r.Players,
			"status":   r.Status,
			"turn":     r.board.Turn,
			"player_x": r.board.PlayerX,
			"player_o": r.board.PlayerO,
		}
		_ = r.store.SaveRoom(ctx, r.ID, roomData, 30*time.Minute)
	}

	// 🎮 Notify both players who starts
	_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "start_game", map[string]any{
		"room_id":     r.ID,
		"first_turn":  firstTurn,
		"player_x":    r.board.PlayerX,
		"player_o":    r.board.PlayerO,
		"toss_winner": firstTurn,
	})
	return nil
}

// MakeMove applies a player move and syncs Redis + Postgres data.
func (r *Room) MakeMove(ctx context.Context, playerID string, index int, elapsed time.Duration, usedGrace bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	board, err := r.store.LoadBoard(ctx, r.ID)
	if err == nil && board != nil {
		r.board = board
	}

	delta, err := r.engine.ApplyMove(r.board, playerID, index, elapsed, usedGrace)
	if err != nil {
		return err
	}
	r.scores[playerID] += delta

	// ✅ Persist board to Redis (retry safe)
	for i := 0; i < 3; i++ {
		if err := r.store.SaveBoard(ctx, r.ID, r.board); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// ✅ Sync latest room info in Redis (turn, players)
	if r.store != nil {
		roomData := map[string]interface{}{
			"room_id":  r.ID,
			"players":  r.Players,
			"status":   r.Status,
			"turn":     r.board.Turn,
			"player_x": r.board.PlayerX,
			"player_o": r.board.PlayerO,
		}
		_ = r.store.SaveRoom(ctx, r.ID, roomData, 30*time.Minute)
	}

	// ✅ Broadcast move to both players
	_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "move_made", map[string]any{
		"board":  r.board,
		"scores": r.scores,
	})

	// ✅ Handle game end
	if r.board.Finished {
		r.Status = "completed"

		winner := r.board.Winner // already holds the playerID of winner

		// Save summary & scores in Postgres
		_ = r.store.SaveGameSummary(ctx, r.ID, winner)
		for pid, sc := range r.scores {
			_ = r.store.SavePlayerScore(ctx, r.ID, pid, sc)
		}

		leaderboard, _ := r.store.GetLeaderboard(ctx, "tic-tac-toe")

		// ✅ Update room state in Redis (shorter TTL after completion)
		if r.store != nil {
			roomData := map[string]interface{}{
				"room_id":  r.ID,
				"players":  r.Players,
				"status":   r.Status,
				"winner":   winner,
				"ended_at": time.Now().Format(time.RFC3339),
			}
			_ = r.store.SaveRoom(ctx, r.ID, roomData, 10*time.Minute)
		}

		// Broadcast game over
		_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "game_over", map[string]any{
			"board":       r.board,
			"scores":      r.scores,
			"leaderboard": leaderboard,
		})
		// 🎯 Auto-return players to group chat after short delay
		go func(roomID string) {
			time.Sleep(10 * time.Second)
			msg := map[string]any{
				"message": "Game finished! Returning to group chat...",
			}
			// Broadcast to all players in this room
			_ = r.broadcaster.BroadcastToRoom(context.Background(), roomID, "return_to_group_chat", msg)
		}(r.ID)

	}

	return nil
}
