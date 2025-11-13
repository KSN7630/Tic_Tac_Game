package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appErr "github.com/KSN7630/Tic_Tac_Game/backend/internal/errors"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisStore handles ephemeral data (board, chat, replay)
type RedisStore struct {
	Rdb *redis.Client
}

// NewRedisStore creates a connected Redis client
func NewRedisStore(addr, password string, db int) *RedisStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisStore{Rdb: rdb}
}

// SaveBoard persists board state with TTL
func (r *RedisStore) SaveBoard(ctx context.Context, gameID string, b *models.BoardState) error {
	data, err := json.Marshal(b)

	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}

	if err := r.Rdb.Set(ctx, "board:"+gameID, data, 30*time.Minute).Err(); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}

	return nil
}

func (r *RedisStore) LoadBoard(ctx context.Context, gameID string) (*models.BoardState, error) {
	val, err := r.Rdb.Get(ctx, "board:"+gameID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, appErr.New(appErr.ErrNotFound, "board data not found in redis")
		}
		return nil, appErr.New(appErr.ErrRedisReadFailed, err.Error())
	}
	var b models.BoardState
	if err := json.Unmarshal(val, &b); err != nil {
		return nil, appErr.New(appErr.ErrDataDecodingFailed, err.Error())
	}
	return &b, nil
}

// SaveChatMessage stores message in Redis List with TTL
func (r *RedisStore) SaveChatMessage(ctx context.Context, key string, msg interface{}, ttl time.Duration) error {
	data, _ := json.Marshal(msg)
	pipe := r.Rdb.Pipeline()
	pipe.RPush(ctx, key, data)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}

	return nil
}

// LoadChatMessages returns all chat messages
func (r *RedisStore) LoadChatMessages(ctx context.Context, key string) ([]string, error) {
	msgs, err := r.Rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, appErr.New(appErr.ErrRedisReadFailed, err.Error())
	}
	return msgs, nil
}

// SaveReplayMove stores move for short time replay feature
func (r *RedisStore) SaveReplayMove(ctx context.Context, gameID string, move interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("replay:%s", gameID)

	data, err := json.Marshal(move)
	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}

	pipe := r.Rdb.Pipeline()
	pipe.RPush(ctx, key, data)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}
	return nil
}

// SaveUser stores guest user information with TTL (e.g. 30 minutes)
func (r *RedisStore) SaveUser(ctx context.Context, player models.Player, ttl time.Duration) error {
	data, err := json.Marshal(player)
	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}
	key := fmt.Sprintf("user:%s", player.ID)
	if err := r.Rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}
	return nil
}

// SaveRoom stores a room snapshot (players, status, etc.) with TTL
func (r *RedisStore) SaveRoom(ctx context.Context, roomID string, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}
	key := fmt.Sprintf("room:%s", roomID)
	if err := r.Rdb.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}
	return nil
}

func (r *RedisStore) SaveGroupChat(ctx context.Context, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}
	pipe := r.Rdb.Pipeline()
	key := "chat:group"
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, 999)
	pipe.Expire(ctx, key, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}
	return nil
}

func (r *RedisStore) SaveMatchChat(ctx context.Context, roomID string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return appErr.New(appErr.ErrDataEncodingFailed, err.Error())
	}
	pipe := r.Rdb.Pipeline()
	key := fmt.Sprintf("chat:match:%s", roomID)
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, 200)
	pipe.Expire(ctx, key, 30*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		return appErr.New(appErr.ErrRedisWriteFailed, err.Error())
	}

	return nil
}

func (r *RedisStore) Close() error {
	return r.Rdb.Close()
}
