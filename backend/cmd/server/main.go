package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/room"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

func main() {
	ctx := context.Background()

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	//Redis setup
	redisStore := storage.NewRedisStore("localhost:6379", "", 0)
	defer redisStore.Close()

	// Initialize scoring and game engines
	scorer := scoring.New()
	engine := game.NewEngine(scorer)

	// Setup WebSocket broadcaster
	netBC := network.NewNetBroadcaster()

	// Create Lobby (connects Redis, broadcaster, and engines)
	l := lobby.NewLobby(redisStore, netBC, engine, scorer)

	// Default group channel - room-lobby
	r := room.NewRoom("room-lobby", redisStore, netBC, engine, scorer)
	l.AddRoom(ctx, "room-lobby", r)

	//  WebSocket route setup
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", network.WSHandler(l, netBC))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// --- START SERVER (in goroutine) ---
	go func() {
		log.Println("Server started at :8080 (Redis connected)")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe Error: %v", err)
		}
	}()

	// --- GRACEFUL SHUTDOWN HANDLER ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // wait for interrupt
	log.Println(" Shutdown signal received...")

	// Timeout context for shutdown
	ctxShutdown, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server gracefully stopped")
}
