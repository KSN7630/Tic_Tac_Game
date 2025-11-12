package events

import (
	"context"

	"github.com/gorilla/websocket"
)

// Broadcaster defines methods for sending messages to connected clients.
// It’s implemented by network.NetBroadcaster.
type Broadcaster interface {
	RegisterConn(roomID string, conn *websocket.Conn)
	UnregisterConn(roomID string, conn *websocket.Conn)
	BroadcastToRoom(ctx context.Context, roomID, event string, payload any) error
	BroadcastToAll(payload any) error
}
