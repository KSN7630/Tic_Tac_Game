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

	// Redis (ephemeral)
	redisStore := storage.NewRedisStore("localhost:6379", "", 0)
	defer redisStore.Close()

	// Postgres (persistent)
	pgStore, err := storage.NewPostgresStore(ctx, "postgres://postgres:775688@localhost:5432/tictactoe?sslmode=disable")

	if err != nil {
		log.Fatalf("Postgres connection failed: %v", err)
	}
	defer pgStore.Close()

	// Setup game and scoring engines
	scorer := scoring.New()
	engine := game.NewEngine(scorer)

	// Setup network broadcaster and lobby
	netBC := network.NewNetBroadcaster()
	l := lobby.NewLobby(redisStore, netBC)

	// Create demo room for testing
	r := room.NewRoom("room-1", redisStore, netBC, engine, scorer)
	l.CreateRoom(ctx, "room-1", r)

	// Start WebSocket server
	http.HandleFunc("/ws", network.WSHandler(l, netBC, nil))

	log.Println("✅ Server started at :8080 (Redis + Postgres connected)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
