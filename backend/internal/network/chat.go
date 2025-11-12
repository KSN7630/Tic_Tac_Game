package network

import (
	"context"
	"log"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// ChatMessage represents a group chat message payload.
type ChatMessage struct {
	From     string    `json:"from"`
	Username string    `json:"username"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}

// SaveAndBroadcastGroupChat saves a group chat message to Redis (24h TTL)
// and broadcasts it globally to all connected WebSocket clients.
func SaveAndBroadcastGroupChat(ctx context.Context, store storage.Storage, bc *NetBroadcaster, msg ChatMessage) {
	// Store in Redis (persist for 24h)
	payload := map[string]any{
		"from":     msg.From,
		"username": msg.Username,
		"message":  msg.Message,
		"at":       msg.At.Format(time.RFC3339),
	}

	if err := store.SaveGroupChat(ctx, payload); err != nil {
		log.Printf("⚠️ failed to save group chat: %v", err)
	}

	// Broadcast globally
	if err := bc.BroadcastToAll(map[string]any{
		"event":   "group_chat",
		"payload": payload,
	}); err != nil {
		log.Printf("⚠️ failed to broadcast group chat: %v", err)
	}
}
