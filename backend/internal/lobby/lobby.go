package lobby

import (
	"context"
	"fmt"
	"sync"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/events"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/room"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// Lobby manages multiple rooms and matchmaking.
type Lobby struct {
	mu          sync.RWMutex
	rooms       map[string]*room.Room
	store       storage.Storage
	broadcaster events.Broadcaster
	engine      *game.GameEngine
	scorer      *scoring.ScoringEngine
}

// NewLobby returns a lobby instance.
// Provide storage, broadcaster, engine and scorer (engine/scorer can be nil in tests).
func NewLobby(store storage.Storage, bc events.Broadcaster, eng *game.GameEngine, sc *scoring.ScoringEngine) *Lobby {
	return &Lobby{
		rooms:       make(map[string]*room.Room),
		store:       store,
		broadcaster: bc,
		engine:      eng,
		scorer:      sc,
	}
}

// AddRoom (alias of CreateRoom) creates a new room and stores it in lobby.
func (l *Lobby) AddRoom(ctx context.Context, id string, r *room.Room) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rooms[id] = r
}

// GetRoom fetches a room by id.
func (l *Lobby) GetRoom(id string) (*room.Room, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	r, ok := l.rooms[id]
	return r, ok
}

// Store returns the storage backend.
func (l *Lobby) Store() storage.Storage {
	return l.store
}

// Broadcast returns the broadcaster.
func (l *Lobby) Broadcaster() events.Broadcaster {
	return l.broadcaster
}

// CreateChallenge creates a new challenge (room) in waiting state.
func (l *Lobby) CreateChallenge(ctx context.Context, challenger models.Player, message string) (string, error) {
	// generate a stable unique room id
	roomID := fmt.Sprintf("room-%d", time.Now().UnixNano())

	// create room using lobby's store, broadcaster, engine and scorer
	rm := room.NewRoom(roomID, l.store, l.broadcaster, l.engine, l.scorer)

	// add challenger as first player
	if err := rm.AddPlayer(challenger); err != nil {
		// If AddPlayer already returned an AppError → return
		if _, ok := err.(appErr.AppError); ok {
			return "", err
		}
		// Otherwise wrap unexpected error
		return "", appErr.New(appErr.ErrRoomCreateFailed, err.Error())
	}

	// register in lobby
	l.mu.Lock()
	l.rooms[roomID] = rm
	l.mu.Unlock()

	// persist room snapshot to Redis (best-effort)
	if l.store != nil {
		roomData := map[string]interface{}{
			"room_id": roomID,
			"players": rm.Players,
			"status":  "waiting",
			"created": time.Now().Format(time.RFC3339),
		}
		_ = l.store.SaveRoom(ctx, roomID, roomData, 30*time.Minute)
	}

	// broadcast new challenge to group (payload includes event type)
	if l.broadcaster != nil {
		// send a generic payload; receiver can inspect payload.event or payload.type
		_ = l.broadcaster.BroadcastToAll(map[string]any{
			"event": "challenge_created",
			"payload": map[string]any{
				"room_id":    roomID,
				"challenger": challenger,
				"message":    message,
			},
		})
	}

	return roomID, nil
}

// AcceptChallenge lets a second player join and start the match.
func (l *Lobby) AcceptChallenge(ctx context.Context, roomID string, accepter models.Player) error {
	rm, ok := l.GetRoom(roomID)
	if !ok {
		return appErr.New(appErr.ErrRoomNotFound, "requested room does not exist")
	}

	// add accepter as second player
	if err := rm.AddPlayer(accepter); err != nil {
		// If AddPlayer already returned an AppError → return
		if _, ok := err.(appErr.AppError); ok {
			return err
		}
		// Otherwise wrap unexpected error
		return appErr.New(appErr.ErrChallengeAccept, err.Error())
	}

	// update persisted room
	if l.store != nil {
		roomData := map[string]interface{}{
			"room_id": roomID,
			"players": rm.Players,
			"status":  "in_progress",
		}
		_ = l.store.SaveRoom(ctx, roomID, roomData, 30*time.Minute)
	}

	// notify players in the room
	if l.broadcaster != nil {
		_ = l.broadcaster.BroadcastToRoom(ctx, roomID, "challenge_accepted", map[string]any{
			"room_id": roomID,
			"players": rm.Players,
		})
	}

	// start the game (this will do toss, save board, broadcast start_game, etc.)
	return rm.StartGame(ctx)
}
