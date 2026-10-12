package room

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
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
	LastMoveAt  time.Time
}

// NewRoom creates a new game room.
func NewRoom(id string, store storage.Storage, bc events.Broadcaster, eng *game.GameEngine, sc *scoring.ScoringEngine) *Room {
	r := &Room{
		ID:          id,
		Status:      "waiting",
		store:       store,
		broadcaster: bc,
		engine:      eng,
		scorer:      sc,
		scores:      make(map[string]int),
		board:       &models.BoardState{GameID: id, Cells: [9]string{}, Created: time.Now()},
	}
	return r
}

// AddPlayer adds a player to the room and updates Redis.
func (r *Room) AddPlayer(p models.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Players) >= 2 {
		return appErr.New(appErr.ErrRoomFull, "room already has 2 players")
	}
	for _, existing := range r.Players {
		if existing.ID == p.ID {
			return nil
		}
	}

	r.Players = append(r.Players, p)
	// ensure score entry exists
	if _, ok := r.scores[p.ID]; !ok {
		r.scores[p.ID] = 0
	}

	// Save room info in Redis after player joins
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
		return appErr.New(appErr.ErrNotEnoughPlayers, "need 2 players to start the game")
	}

	r.Status = "in_progress"

	// Toss logic — randomly decide who starts (fair chance).
	// The global rand source is auto-seeded in Go 1.20+, so no manual Seed.
	startIndex := rand.Intn(2) // 0 or 1

	firstTurn := r.Players[startIndex].ID
	secondPlayer := r.Players[1-startIndex].ID

	// Initialize board with PlayerX (first) and PlayerO (second)
	r.board = models.NewEmptyBoard(r.ID, firstTurn)
	r.board.PlayerX = firstTurn
	r.board.PlayerO = secondPlayer
	r.board.Turn = firstTurn // PlayerX starts first
	r.LastMoveAt = time.Now()

	// Ensure scores map has entries for both players
	if _, ok := r.scores[firstTurn]; !ok {
		r.scores[firstTurn] = 0
	}
	if _, ok := r.scores[secondPlayer]; !ok {
		r.scores[secondPlayer] = 0
	}

	// Save initial board state in Redis
	if r.store != nil {
		if err := r.store.SaveBoard(ctx, r.ID, r.board); err != nil {
			return appErr.New(appErr.ErrSaveBoardFailed, err.Error())

		}
		// Update room state in Redis
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

	// Notify both players who starts
	if r.broadcaster != nil {
		_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "start_game", map[string]any{
			"room_id":     r.ID,
			"first_turn":  firstTurn,
			"player_x":    r.board.PlayerX,
			"player_o":    r.board.PlayerO,
			"toss_winner": firstTurn,
		})
	}

	return nil
}

// MakeMove applies a player's move and syncs Redis data (no Postgres).
func (r *Room) MakeMove(ctx context.Context, playerID string, index int, usedGrace bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load latest board from Redis (if available)
	if r.store != nil {
		board, err := r.store.LoadBoard(ctx, r.ID)
		if err == nil && board != nil {
			// Restore missing fields
			if board.PlayerX == "" && len(r.Players) > 0 {
				board.PlayerX = r.Players[0].ID
			}
			if board.PlayerO == "" && len(r.Players) > 1 {
				board.PlayerO = r.Players[1].ID
			}
			if board.Turn == "" {
				board.Turn = board.PlayerX
			}
			r.board = board
		}
	}

	// Defensive initialization
	if r.board == nil {
		first := ""
		if len(r.Players) > 0 {
			first = r.Players[0].ID
		}
		r.board = models.NewEmptyBoard(r.ID, first)
		if r.store != nil {
			_ = r.store.SaveBoard(ctx, r.ID, r.board)
		}
	}

	// Log for debugging
	log.Printf("➡️ MakeMove: room=%s player=%s index=%d turn=%s",
		r.ID, playerID, index, r.board.Turn)

	// Apply move
	elapsed := time.Since(r.LastMoveAt)

	delta, err := r.engine.ApplyMove(r.board, playerID, index, elapsed, usedGrace)

	if err != nil {
		log.Printf("ApplyMove failed: %v", err)
		if _, ok := appErr.As(err); ok {
			// it's already structured → bubble up
			return err
		}

		// unexpected error → wrap into internal error type
		return appErr.New(appErr.ErrInvalidBoardState, err.Error())
	}

	// Update score
	if _, ok := r.scores[playerID]; !ok {
		r.scores[playerID] = 0
	}
	r.scores[playerID] += delta
	r.LastMoveAt = time.Now()

	// Save replay move to Redis
	if r.store != nil {
		_ = r.store.SaveReplayMove(ctx, r.ID, models.Move{
			PlayerID: playerID,
			Index:    index,
			At:       time.Now(),
		}, 30*time.Minute)
	}

	// Save board to Redis
	if r.store != nil {
		_ = r.store.SaveBoard(ctx, r.ID, r.board)
	}

	// Save room metadata
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

	// Broadcast move
	if r.broadcaster != nil {
		_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "move_made", map[string]any{
			"board":  r.board,
			"scores": r.scores,
		})
	}

	// GAME OVER LOGIC (NO POSTGRES)
	if r.board.Finished {
		r.Status = "completed"
		winner := r.board.Winner // playerID of winner or empty for draw

		// Update room state
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

		// Broadcast game over (leaderboard empty)
		if r.broadcaster != nil {
			_ = r.broadcaster.BroadcastToRoom(ctx, r.ID, "game_over", map[string]any{
				"board":       r.board,
				"scores":      r.scores,
				"leaderboard": []string{}, // No postgres -> empty
			})
		}

		// Notify room to return to chat
		if r.broadcaster != nil {
			go func(roomID string) {
				time.Sleep(10 * time.Second)
				_ = r.broadcaster.BroadcastToRoom(
					context.Background(),
					roomID,
					"return_to_group_chat",
					map[string]any{"message": "Game finished! Returning to chat."},
				)
			}(r.ID)
		}
	}

	return nil
}
