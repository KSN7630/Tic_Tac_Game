package network

import (
	"context"
	"log"
	"sync"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
	"github.com/gorilla/websocket"
)

// NetBroadcaster implements events.Broadcaster
// and manages all websocket connections grouped by room.
type NetBroadcaster struct {
	connRegistry map[string]map[*websocket.Conn]struct{}
	mu           sync.RWMutex
}

// NewNetBroadcaster creates a new NetBroadcaster.
func NewNetBroadcaster() *NetBroadcaster {
	return &NetBroadcaster{
		connRegistry: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// connWriteLocks serializes writes per websocket connection.
// gorilla/websocket only supports one concurrent writer per connection, but a
// single connection is written to from many goroutines (its own read loop plus
// every broadcast triggered by other players). Without this, two simultaneous
// writes corrupt the connection and the client is disconnected.
var connWriteLocks sync.Map // *websocket.Conn -> *sync.Mutex

func connLock(conn *websocket.Conn) *sync.Mutex {
	m, _ := connWriteLocks.LoadOrStore(conn, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// safeWriteJSON writes a JSON message to conn while holding that connection's
// write lock, guaranteeing no two goroutines write concurrently.
func safeWriteJSON(conn *websocket.Conn, v any) error {
	mu := connLock(conn)
	mu.Lock()
	defer mu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteJSON(v)
}

// releaseConnLock drops the write lock once a connection is gone, so the
// sync.Map does not grow without bound.
func releaseConnLock(conn *websocket.Conn) {
	connWriteLocks.Delete(conn)
}

// RegisterConn adds a connection to a specific room.
func (nb *NetBroadcaster) RegisterConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if nb.connRegistry[roomID] == nil {
		nb.connRegistry[roomID] = make(map[*websocket.Conn]struct{})
	}
	nb.connRegistry[roomID][conn] = struct{}{}
	log.Printf("Registered connection for room %s (total %d connections)", roomID, len(nb.connRegistry[roomID]))
}

// UnregisterConn removes a connection from a specific room and closes it.
func (nb *NetBroadcaster) UnregisterConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if conns, ok := nb.connRegistry[roomID]; ok {
		delete(conns, conn)
		_ = conn.Close()
		if len(conns) == 0 {
			delete(nb.connRegistry, roomID)
		}
		log.Printf("Unregistered connection for room %s (remaining %d)", roomID, len(conns))
	}
	releaseConnLock(conn)
}

// BroadcastToRoom sends an event + payload to all connections in the given room.
func (nb *NetBroadcaster) BroadcastToRoom(ctx context.Context, roomID string, event string, payload any) error {
	// Snapshot connections so we don't hold the lock during network writes.
	nb.mu.RLock()
	conns := nb.connRegistry[roomID]
	targets := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		targets = append(targets, conn)
	}
	nb.mu.RUnlock()

	if len(targets) == 0 {
		return nil
	}

	msg := map[string]any{"event": event, "payload": payload}

	var lastErr error
	for _, conn := range targets {
		if err := safeWriteJSON(conn, msg); err != nil {
			log.Printf(" broadcast error to room %s: %v", roomID, err)
			go nb.removeConn(roomID, conn)
			lastErr = err
			// Keep going so the remaining players still receive the event.
		}
	}

	if lastErr != nil {
		return appErr.New(appErr.ErrBroadcastFailed, lastErr.Error())
	}
	return nil
}

// BroadcastToAll sends a message globally to all connections across all rooms.
// This is used for group chat and lobby-wide notifications. Callers already
// pass a full {"event": ..., "payload": ...} envelope (see lobby.CreateChallenge
// and the send_group_chat handler) — do not re-wrap it here, or the client
// receives a doubly-nested payload with the real event/fields one level too deep.
func (nb *NetBroadcaster) BroadcastToAll(payload any) error {
	msg := payload

	// Deduplicate: a single connection may be registered under several rooms
	// (e.g. lobby + game room), and must receive each message exactly once.
	targets := make(map[*websocket.Conn]string)
	nb.mu.RLock()
	for roomID, conns := range nb.connRegistry {
		for conn := range conns {
			if _, seen := targets[conn]; !seen {
				targets[conn] = roomID
			}
		}
	}
	nb.mu.RUnlock()

	var lastErr error
	for conn, roomID := range targets {
		if err := safeWriteJSON(conn, msg); err != nil {
			log.Printf("global broadcast error (room %s): %v", roomID, err)
			go nb.removeConn(roomID, conn)
			lastErr = err
		}
	}

	if lastErr != nil {
		return appErr.New(appErr.ErrBroadcastFailed, lastErr.Error())
	}
	return nil
}

// removeConn safely deletes a bad connection from the registry.
func (nb *NetBroadcaster) removeConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if conns, ok := nb.connRegistry[roomID]; ok {
		delete(conns, conn)
		_ = conn.Close()
		if len(conns) == 0 {
			delete(nb.connRegistry, roomID)
		}
	}
}
