package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/handlers/models"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ------------------------- Configurable Upgrader -------------------------

// upgrader is assigned at runtime by InitializeUpgrader.
var upgrader websocket.Upgrader

// InitializeUpgrader configures origin checking based on environment.
// env: "dev" (allow all) or "prod" (restrict to allowedOrigins).
func InitializeUpgrader(env string, allowedOrigins []string) {
	if env == "prod" && len(allowedOrigins) > 0 {
		upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				for _, a := range allowedOrigins {
					if origin == a {
						return true
					}
				}
				return false
			},
		}
	} else {
		// default: permissive for local/dev
		upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
	}
	log.Printf("WebSocket upgrader initialized for env=%s\n", env)
}

// ------------------------- Types & Globals -------------------------

// Client represents a connected websocket client.
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte // buffered outbound channel
}

// All active clients (concurrent-safe)
var clients sync.Map // key: *Client, value: bool

// gamePlayers maps gameID -> *sync.Map (key: *Client, value: bool)
// This allows concurrent add/remove per game without global locking.
var gamePlayers sync.Map // key: string => value: *sync.Map

// broadcast channel for group-wide messages
var broadcast = make(chan []byte, 1024)

// ------------------------- Game player management -------------------------

// addClientToGame registers a client into a game's client set.
func addClientToGame(gameID string, c *Client) {
	val, _ := gamePlayers.LoadOrStore(gameID, &sync.Map{})
	pm := val.(*sync.Map)
	pm.Store(c, true)
}

// removeClientFromAllGames removes a client from every game's set it is present in.
// If a game's set becomes empty, deletes the game entry to avoid leaks.
func removeClientFromAllGames(c *Client) {
	gamePlayers.Range(func(key, val interface{}) bool {
		gameID := key.(string)
		pm := val.(*sync.Map)

		pm.Delete(c)

		// Check if pm is empty; if so, delete the game entry.
		empty := true
		pm.Range(func(_, _ interface{}) bool {
			empty = false
			return false // stop ranging early
		})
		if empty {
			gamePlayers.Delete(gameID)
		}
		return true
	})
}

// ------------------------- Broadcast functions -------------------------

// handleBroadcast fans out group messages to all connected clients in a non-blocking way.
func handleBroadcast() {
	for msg := range broadcast {
		clients.Range(func(key, _ interface{}) bool {
			client := key.(*Client)
			select {
			case client.Send <- msg:
				// queued ok
			default:
				// slow client, remove it
				log.Println("Dropping slow client (group broadcast)")
				removeClient(client)
			}
			return true
		})
	}
}

// broadcastToGame sends msg only to clients registered in gamePlayers[gameID].
func broadcastToGame(gameID string, msg []byte) {
	val, ok := gamePlayers.Load(gameID)
	if !ok {
		// no players in game (maybe not registered yet)
		return
	}
	pm := val.(*sync.Map)
	pm.Range(func(key, _ interface{}) bool {
		client := key.(*Client)
		select {
		case client.Send <- msg:
			// queued
		default:
			// slow/blocked client: remove it both globally and from games
			log.Printf("Dropping slow client in game %s\n", gameID)
			removeClient(client)
		}
		return true
	})
}

// ------------------------- Client lifecycle helpers -------------------------

// removeClient removes client from global map, all games and closes resources.
// Safe to call multiple times (uses recover for double-close).
func removeClient(c *Client) {
	clients.Delete(c)
	// remove from any game lists
	removeClientFromAllGames(c)

	// close send channel & websocket (recover from double close)
	defer func() {
		if r := recover(); r != nil {
			// ignore panic from double close
		}
	}()
	close(c.Send)
	c.Conn.Close()
}

// ------------------------- Per-client pumps -------------------------

// readPump reads messages from the websocket for a single client.
func (c *Client) readPump() {
	defer removeClient(c)

	// Optionally set read limits and deadlines here (for production)
	// c.Conn.SetReadLimit(512*1024)

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			// normal close or network error
			log.Println("read error:", err)
			break
		}

		var msg models.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Println("invalid json from client:", err)
			continue
		}
		// stamp server timestamp
		msg.Timestamp = time.Now().Unix()

		switch msg.Type {
		case "group_chat":
			// persist & broadcast to everyone
			if err := SaveGroupChatMessage(msg); err != nil {
				log.Println("redis save (group) error:", err)
			}
			// send raw JSON to broadcast channel (non-blocking)
			enc, _ := json.Marshal(msg)
			select {
			case broadcast <- enc:
			default:
				log.Println("broadcast full - dropping group message")
			}

		case "match_chat":
			// ensure client is registered in this game (idempotent)
			if msg.GameID != "" {
				addClientToGame(msg.GameID, c)
			}
			// persist to match list and send only to that game's clients
			if err := SaveMatchChatMessage(msg); err != nil {
				log.Println("redis save (match) error:", err)
			}
			enc, _ := json.Marshal(msg)
			// broadcast to players in that game (async but immediate)
			go broadcastToGame(msg.GameID, enc)

		default:
			log.Println("unknown message type:", msg.Type)
		}
	}
}

// writePump writes outbound messages (pushed into c.Send) to the websocket.
func (c *Client) writePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("write error:", err)
			break
		}
	}
	// when send channel closed or write error -> ensure removal
	removeClient(c)
}

// ------------------------- Connection handler & startup -------------------------

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	if upgrader.CheckOrigin == nil {
		InitializeUpgrader("dev", nil)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	clients.Store(client, true)

	// ✅ if gameID exists in context, register this client immediately
	if gameID, ok := r.Context().Value("game_id").(string); ok && gameID != "" {
		addClientToGame(gameID, client)
		log.Printf("✅ Client %s joined game %s\n", client.ID, gameID)
	}

	log.Println("✅ New WebSocket client connected")

	go client.writePump()
	client.readPump()
}

// StartWebsocketDispatcher should be called once at server startup.
func StartWebsocketDispatcher() {
	go handleBroadcast()
}

// HandleGroupChat handles the /ws/group route for global chat
func HandleGroupChat(w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket connection for group chat")
	HandleConnections(w, r)
}

// HandleGameChat handles the /ws/game/{game_id} route for match-specific chat
func HandleGameChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["game_id"]
	if gameID == "" {
		http.Error(w, "missing game_id", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket connection for game: %s\n", gameID)

	// Inject game_id in request context (so HandleConnections can use it if needed)
	ctx := context.WithValue(r.Context(), "game_id", gameID)
	r = r.WithContext(ctx)

	HandleConnections(w, r)
}
