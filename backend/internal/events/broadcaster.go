package events

import "context"

// Broadcaster defines the minimal methods the network layer must implement
// so that game/room packages can broadcast events without importing the network.
type Broadcaster interface {
	// Broadcast to all clients in a room (game update, scoreboard, move)
	BroadcastToRoom(ctx context.Context, roomID string, event string, payload any) error

	// Broadcast globally (group chat, server announcements)
	BroadcastGlobal(ctx context.Context, event string, payload any) error
}
