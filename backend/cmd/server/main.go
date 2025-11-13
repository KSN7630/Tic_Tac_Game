package main

import (
	"context"
	"log"
	"net/http"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/room"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

func main() {
	ctx := context.Background()

	// ✅ 1️⃣ Redis (ephemeral, game & chat state)
	redisStore := storage.NewRedisStore("localhost:6379", "", 0)
	defer redisStore.Close()

	// ✅ 3️⃣ Initialize scoring and game engines
	scorer := scoring.New()
	engine := game.NewEngine(scorer)

	// ✅ 4️⃣ Setup WebSocket broadcaster
	netBC := network.NewNetBroadcaster()

	// ✅ 5️⃣ Create Lobby (connects Redis, broadcaster, and engines)
	l := lobby.NewLobby(redisStore, netBC, engine, scorer)

	// ✅ 6️⃣ (Optional) Create a demo room for testing or lobby presence
	r := room.NewRoom("room-lobby", redisStore, netBC, engine, scorer)
	l.AddRoom(ctx, "room-lobby", r)

	// ✅ 7️⃣ WebSocket route
	http.HandleFunc("/ws", network.WSHandler(l, netBC))

	// ✅ 8️⃣ Start HTTP server
	log.Println("✅ Server started at :8080 (Redis + Postgres connected)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
