package network

import (
	"context"
	"log"
	"sync"
	"time"

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

// RegisterConn adds a connection to a specific room.
func (nb *NetBroadcaster) RegisterConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if nb.connRegistry[roomID] == nil {
		nb.connRegistry[roomID] = make(map[*websocket.Conn]struct{})
	}
	nb.connRegistry[roomID][conn] = struct{}{}
	log.Printf("✅ Registered connection for room %s (total %d connections)", roomID, len(nb.connRegistry[roomID]))
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
		log.Printf("❎ Unregistered connection for room %s (remaining %d)", roomID, len(conns))
	}
}

// BroadcastToRoom sends an event + payload to all connections in the given room.
func (nb *NetBroadcaster) BroadcastToRoom(ctx context.Context, roomID string, event string, payload any) error {
	nb.mu.RLock()
	conns, ok := nb.connRegistry[roomID]
	nb.mu.RUnlock()

	if !ok || len(conns) == 0 {
		return nil
	}

	msg := map[string]any{"event": event, "payload": payload}

	for conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second)) // prevents blocking on slow clients
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("⚠️ broadcast error to room %s: %v", roomID, err)
			go nb.removeConn(roomID, conn)
		}
	}

	return nil
}

// BroadcastToAll sends a payload globally to all connections across all rooms.
// This is used for group chat and lobby-wide notifications.
func (nb *NetBroadcaster) BroadcastToAll(payload any) error {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	msg := map[string]any{"event": "group_chat", "payload": payload}

	for roomID, conns := range nb.connRegistry {
		for conn := range conns {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("⚠️ global broadcast error (room %s): %v", roomID, err)
				go nb.removeConn(roomID, conn)
			}
		}
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
