package game

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/models"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

// StartTossPhase runs the settle + toss logic asynchronously for a given room.
// It waits a few seconds, chooses a random first player, updates Redis,
// and broadcasts only to players connected in that game.
func StartTossPhase(roomID string) {
	log.Printf("Toss phase started for room %s", roomID)

	// 1️ Simulate small settle delay before toss
	time.Sleep(3 * time.Second)

	// 2️ Load the latest room data from Redis
	ctx := storage.Ctx
	data, err := storage.Rdb.Get(ctx, "room:"+roomID).Bytes()
	if err != nil {
		log.Println("Failed to fetch room for toss:", err)
		return
	}

	var room models.RoomWithTurn
	if err := json.Unmarshal(data, &room); err != nil {
		log.Println("Unmarshal error:", err)
		return
	}

	// 3️ Randomly pick which player goes first
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(2) == 0 {
		room.Turn = room.PlayerX
	} else {
		room.Turn = room.PlayerO
	}
	room.FirstTurn = room.Turn
	room.Status = "in_progress"

	// 4️ Save updated state back to Redis
	updated, _ := json.Marshal(room)
	if err := storage.Rdb.Set(ctx, "room:"+room.RoomID, updated, 30*time.Minute).Err(); err != nil {
		log.Println("Failed to update room:", err)
	}

	// 5️ Broadcast toss result ONLY to players in this game
	network.BroadcastGameEvent(room.RoomID, "toss_result", map[string]string{
		"room_id":      room.RoomID,
		"first_player": room.Turn,
	})
	log.Printf("Toss result: %s starts first in %s", room.Turn, room.RoomID)

	// 6 After short pause, broadcast start of first turn (again, only to this game)
	time.Sleep(1 * time.Second)
	network.BroadcastGameEvent(room.RoomID, "start_turn", map[string]string{
		"room_id": room.RoomID,
		"turn":    room.Turn,
	})
	// Initialize empty board in Redis for this match
	board := NewBoard(room.PlayerX, room.PlayerO, room.Turn)
	if err := SaveBoard(room.RoomID, board); err != nil {
		log.Println(" Failed to save board:", err)
	}
}
