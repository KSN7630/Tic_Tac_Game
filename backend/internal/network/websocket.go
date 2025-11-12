package network

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In dev, allow all origins. Restrict in prod.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHandler upgrades HTTP to WebSocket and routes messages.
func WSHandler(l *lobby.Lobby, bc *NetBroadcaster, storeConn func(*websocket.Conn, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Parse query params: ?room=<id>&user=<id>&username=<name>
		q := r.URL.Query()
		roomID := q.Get("room")
		userID := q.Get("user")
		username := q.Get("username")

		if roomID == "" || userID == "" {
			conn.WriteJSON(map[string]any{"error": "room and user query required"})
			return
		}

		ctx := context.Background()
		player := models.Player{
			ID:       userID,
			Username: username,
		}

		// ✅ Save user to Redis with 30-min TTL
		if l != nil && l.Store() != nil {
			if err := l.Store().SaveUser(ctx, player, 30*time.Minute); err != nil {
				log.Printf("failed to save user in redis: %v", err)
			}
		}

		// ✅ Register connection in broadcaster
		bc.RegisterConn(roomID, conn)
		defer bc.UnregisterConn(roomID, conn)

		// ✅ Register player in the room
		rm, ok := l.GetRoom(roomID)
		if !ok {
			conn.WriteJSON(map[string]any{"error": "room not found"})
			return
		}
		if err := rm.AddPlayer(player); err != nil {
			conn.WriteJSON(map[string]any{"error": err.Error()})
			return
		}

		// ✅ WebSocket message read loop
		for {
			var msg map[string]any
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("read error: %v", err)
				break
			}

			action, _ := msg["action"].(string)

			switch action {

			case "start_game":
				ctx := context.Background()
				_ = rm.StartGame(ctx)

			case "make_move":
				idxFloat, _ := msg["index"].(float64)
				idx := int(idxFloat)
				elapsedMs, _ := msg["elapsed_ms"].(float64)
				elapsed := time.Duration(int64(elapsedMs)) * time.Millisecond
				usedGrace, _ := msg["used_grace"].(bool)
				_ = rm.MakeMove(context.Background(), userID, idx, elapsed, usedGrace)

			case "send_group_chat":
				msgText, _ := msg["message"].(string)
				chat := ChatMessage{
					From:    userID,
					Message: msgText,
					At:      time.Now(),
				}
				SaveAndBroadcastGroupChat(context.Background(), l.Store(), bc, chat)

			case "send_match_chat":
				msgText, _ := msg["message"].(string)
				chat := ChatMessage{
					From:    userID,
					Message: msgText,
					At:      time.Now(),
				}
				key := fmt.Sprintf("chat:match:%s", roomID)

				// Save chat in Redis for this room
				if l != nil && l.Store() != nil {
					if err := l.Store().SaveChatMessage(context.Background(), key, chat, 30*time.Minute); err != nil {
						log.Printf("failed to save match chat: %v", err)
					}
				}

				// Broadcast only to this room
				_ = bc.BroadcastToRoom(context.Background(), roomID, "match_chat", map[string]any{
					"from":    chat.From,
					"message": chat.Message,
					"room_id": roomID,
					"at":      chat.At,
				})
			case "get_replay":
				board, err := l.Store().LoadBoard(context.Background(), roomID)
				if err != nil || board == nil {
					conn.WriteJSON(map[string]any{
						"event":   "error",
						"message": "replay not found or expired",
					})
					continue
				}

				// Send the replay data
				conn.WriteJSON(map[string]any{
					"event":   "replay_data",
					"payload": board.Moves,
				})

			default:
				conn.WriteJSON(map[string]any{"error": "unknown action"})
			}
		}
	}
}
