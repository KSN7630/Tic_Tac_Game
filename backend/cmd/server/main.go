package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/config"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/game"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/room"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/scoring"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup Redis
	redisStore := storage.NewRedisStore(cfg.Redis.Host, cfg.Redis.Password, cfg.Redis.DB)
	defer redisStore.Close()

	// Initialize scoring and game engines
	scorer := scoring.New()
	engine := game.NewEngine(scorer)

	// Setup WebSocket broadcaster
	netBC := network.NewNetBroadcaster()

	// Create Lobby (connects Redis, broadcaster, and engines)
	l := lobby.NewLobby(redisStore, netBC, engine, scorer)

	// Default lobby room
	r := room.NewRoom(cfg.Lobby.DefaultRoom, redisStore, netBC, engine, scorer)
	l.AddRoom(ctx, cfg.Lobby.DefaultRoom, r)

	//  WebSocket route setup
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", network.WSHandler(l, netBC))

	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: mux,
	}

	// --- START SERVER (in goroutine) ---

	go func() {
		log.Printf("Server running at %s", cfg.Server.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	// --- GRACEFUL SHUTDOWN HANDLER ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // wait for interrupt
	log.Println(" Shutdown signal received...")

	// Timeout context for shutdown
	ctxShutdown, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server gracefully stopped")
}
