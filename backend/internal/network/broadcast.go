package network

import (
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// NetBroadcaster implements events.Broadcaster and uses the network's connection registry
// to broadcast messages into rooms. This keeps network-specific code inside network package.
type NetBroadcaster struct {
	// connRegistry maps roomIDs to a list of websocket connections.
	// In production this should be more advanced (connection structs, write pumps, etc.)
	connRegistry map[string]map[*websocket.Conn]struct{}
	mu           sync.RWMutex
}

func NewNetBroadcaster() *NetBroadcaster {
	return &NetBroadcaster{
		connRegistry: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (nb *NetBroadcaster) RegisterConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	if nb.connRegistry[roomID] == nil {
		nb.connRegistry[roomID] = make(map[*websocket.Conn]struct{})
	}
	nb.connRegistry[roomID][conn] = struct{}{}
}

func (nb *NetBroadcaster) UnregisterConn(roomID string, conn *websocket.Conn) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	if conns := nb.connRegistry[roomID]; conns != nil {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(nb.connRegistry, roomID)
		}
	}
}

func (nb *NetBroadcaster) BroadcastToRoom(ctx context.Context, roomID string, event string, payload any) error {
	nb.mu.RLock()
	conns := nb.connRegistry[roomID]
	nb.mu.RUnlock()
	for conn := range conns {
		msg := map[string]any{"event": event, "payload": payload}
		if err := conn.WriteJSON(msg); err != nil {
			// Log error and continue; eventually unregister broken connections
			log.Printf("error writing to conn: %v", err)
		}
	}
	return nil
}

func (nb *NetBroadcaster) BroadcastGlobal(ctx context.Context, event string, payload any) error {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	msg := map[string]any{"event": event, "payload": payload}
	for _, conns := range nb.connRegistry {
		for conn := range conns {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("error writing to conn (global): %v", err)
			}
		}
	}
	return nil
}
