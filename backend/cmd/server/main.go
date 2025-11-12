//	entry point

// to run main.go file
// go run cmd/server/main.go -config config/local.yaml
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/config"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/network"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/storage"
	"github.com/gorilla/mux"
)

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")

	fmt.Println("Welcome to Tic_Tac game")

	//load config
	cfg := config.MustLoad()

	//database setup
	storage.InitRedis()
	storage.InitPostgres()
	network.InitializeUpgrader(cfg.Env, []string{"https://your-production-domain.com"})

	// Start broadcast handler goroutine
	network.StartWebsocketDispatcher()

	flag.Parse()

	// Setup Gorilla Mux Router
	router := mux.NewRouter()

	// -------------------- HTTP endpoints --------------------

	// Simple root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Tic Tac Toe Backend!"))
	}).Methods("GET")

	// -------------------- WebSocket endpoints --------------------
	//  Group Chat: global channel (everyone receives messages)
	router.HandleFunc("/ws/group", network.HandleGroupChat).Methods("GET")

	//  Game Chat: one per game (only players in same gameID get messages)
	router.HandleFunc("/ws/game/{game_id}", network.HandleGameChat).Methods("GET")

	//-------------------------Lobby endpoints -------------------------

	router.HandleFunc("/api/user/create", func(w http.ResponseWriter, r *http.Request) {
		user := lobby.CreateGuestUser()
		json.NewEncoder(w).Encode(user)
	}).Methods("POST")

	router.HandleFunc("/api/room/create", func(w http.ResponseWriter, r *http.Request) {
		type req struct {
			PlayerID string `json:"player_id"`
			GameType string `json:"game_type"`
		}
		var body req
		json.NewDecoder(r.Body).Decode(&body)

		room, err := lobby.CreateRoom(body.PlayerID, body.GameType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(room)
	}).Methods("POST")

	router.HandleFunc("/api/room/join/{room_id}", func(w http.ResponseWriter, r *http.Request) {
		roomID := mux.Vars(r)["room_id"]
		type req struct {
			PlayerID string `json:"player_id"`
		}
		var body req
		json.NewDecoder(r.Body).Decode(&body)

		room, err := lobby.JoinRoom(roomID, body.PlayerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(room)
	}).Methods("POST")

	router.HandleFunc("/api/room/{room_id}", func(w http.ResponseWriter, r *http.Request) {
		roomID := mux.Vars(r)["room_id"]
		room, err := lobby.GetRoom(roomID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(room)
	}).Methods("GET")

	//setup server
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}
	fmt.Println("Server started")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	done := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-done

	slog.Info("Sutting down the server")
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	server.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)

}
