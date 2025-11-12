package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// ---------------------- Structs ----------------------

type RoomLobby struct {
	RoomID   string `json:"room_id"`
	PlayerX  string `json:"player_x"`
	PlayerO  string `json:"player_o,omitempty"`
	Status   string `json:"status"` // waiting | active | finished
	GameType string `json:"game_type"`
	Created  int64  `json:"created"`
}

type User struct {
	ID       string `json:"id"`
	JoinedAt int64  `json:"joined_at"`
}

// ---------------------- Redis Keys ----------------------

func roomKey(id string) string { return fmt.Sprintf("room:%s", id) }
func userKey(id string) string { return fmt.Sprintf("user:%s", id) }

// ---------------------- User Management ----------------------

// CreateGuestUser generates a temporary guest user
func CreateGuestUser() User {
	id := fmt.Sprintf("guest_%d", rand.Intn(100000))
	user := User{
		ID:       id,
		JoinedAt: time.Now().Unix(),
	}

	data, _ := json.Marshal(user)
	ctx := context.Background()
	storage.Rdb.Set(ctx, userKey(id), data, 30*time.Minute)
	log.Printf("Created guest user: %s", id)
	return user
}

// ---------------------- Room Management ----------------------

// CreateRoom creates a new game room with PlayerX as the creator
func CreateRoom(playerXID, gameType string) (*RoomLobby, error) {
	roomID := fmt.Sprintf("room_%d", time.Now().UnixNano())
	room := RoomLobby{
		RoomID:   roomID,
		PlayerX:  playerXID,
		Status:   "waiting",
		GameType: gameType,
		Created:  time.Now().Unix(),
	}

	data, err := json.Marshal(room)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := storage.Rdb.Set(ctx, roomKey(roomID), data, 30*time.Minute).Err(); err != nil {
		return nil, err
	}

	log.Printf("Created room: %s by %s", roomID, playerXID)
	network.BroadcastEvent("room_created", room)
	return &room, nil
}

// JoinRoom allows another player (PlayerO) to join an existing room
func JoinRoom(roomID, playerOID string) (*RoomLobby, error) {
	ctx := context.Background()
	val, err := storage.Rdb.Get(ctx, roomKey(roomID)).Result()
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}

	var room RoomLobby
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, err
	}

	if room.PlayerO != "" {
		return nil, fmt.Errorf("room already full")
	}

	room.PlayerO = playerOID
	room.Status = "active"

	data, _ := json.Marshal(room)
	if err := storage.Rdb.Set(ctx, roomKey(roomID), data, 30*time.Minute).Err(); err != nil {
		return nil, err
	}

	log.Printf("Player %s joined room %s", playerOID, roomID)
	network.BroadcastEvent("player_joined", room)

	// If both players are now in, broadcast start signal
	if room.PlayerX != "" && room.PlayerO != "" {
		network.BroadcastEvent("start_game", room)
		go game.StartTossPhase(room.RoomID) //  triggers the toss logic
	}
	return &room, nil
}

// GetRoom retrieves current room state
func GetRoom(roomID string) (*RoomLobby, error) {
	ctx := context.Background()
	val, err := storage.Rdb.Get(ctx, roomKey(roomID)).Result()
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}

	var room RoomLobby
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, err
	}
	return &room, nil
}
