package network

import (
	"context"
	"log"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

type ChatMessage struct {
	From    string    `json:"from"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// SaveAndBroadcastGroupChat now stores chats via RedisStore with 24h TTL
func SaveAndBroadcastGroupChat(ctx context.Context, store storage.Storage, bc *NetBroadcaster, msg ChatMessage) {
	key := "chat:group"
	if err := store.SaveChatMessage(ctx, key, msg, 24*time.Hour); err != nil {
		log.Printf("save group chat failed: %v", err)
	}

	if err := bc.BroadcastGlobal(ctx, "group_chat", msg); err != nil {
		log.Printf("broadcast group chat failed: %v", err)
	}
}
