package network

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// ------------------------- Redis helpers -------------------------

// SaveGroupChatMessage persists a group chat message to Redis (key: chat:group, TTL 24h)
func SaveGroupChatMessage(msg models.Message) error {
	ctx := context.Background()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := "chat:group"
	if err := storage.Rdb.RPush(ctx, key, data).Err(); err != nil {
		return err
	}
	return storage.Rdb.Expire(ctx, key, 24*time.Hour).Err()
}

// SaveMatchChatMessage persists a match chat message to Redis (key: chat:match:<gameID>, TTL 2h)
func SaveMatchChatMessage(msg models.Message) error {
	ctx := context.Background()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("chat:match:%s", msg.GameID)
	if err := storage.Rdb.RPush(ctx, key, data).Err(); err != nil {
		return err
	}
	return storage.Rdb.Expire(ctx, key, 2*time.Hour).Err()
}
