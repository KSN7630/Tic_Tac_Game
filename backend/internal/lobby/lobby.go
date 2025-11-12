package lobby

import (
	"context"
	"sync"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/events"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/room"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// Lobby manages multiple rooms and matchmaking.
type Lobby struct {
	mu    sync.RWMutex
	rooms map[string]*room.Room
	store storage.Storage
	bc    events.Broadcaster
	eng   *room.Room // not used directly; rooms are created via NewRoom
}

// NewLobby returns a lobby instance.
func NewLobby(store storage.Storage, bc events.Broadcaster) *Lobby {
	return &Lobby{
		rooms: make(map[string]*room.Room),
		store: store,
		bc:    bc,
	}
}

// CreateRoom creates a new room and stores it in lobby.
func (l *Lobby) CreateRoom(ctx context.Context, id string, r *room.Room) {
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

func (l *Lobby) Store() storage.Storage {
	return l.store
}
