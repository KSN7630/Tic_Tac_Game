package network

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/KSN7630/Tic_Tac_Game/backend/internal/lobby"
	"github.com/KSN7630/Tic_Tac_Game/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WSHandler manages incoming WebSocket connections.
func WSHandler(l *lobby.Lobby, bc *NetBroadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}
		q := r.URL.Query()
		roomID := q.Get("room")
		userID := q.Get("user")
		username := q.Get("username")

		// ✅ Save connected user in Redis for 30 minutes
		if l != nil && l.Store() != nil {
			ctx := context.Background()
			player := models.Player{ID: userID, Username: username}
			if err := l.Store().SaveUser(ctx, player, 30*time.Minute); err != nil {
				log.Printf("⚠️ failed to save user in redis: %v", err)
			}
		}

		if roomID == "" || userID == "" {
			conn.WriteJSON(map[string]any{"error": "room and user query required"})
			conn.Close()
			return
		}

		bc.RegisterConn(roomID, conn)
		defer func() {
			bc.UnregisterConn(roomID, conn)
			conn.Close()
		}()

		// Ping every 30s to close stale connections
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}()

		for {
			var msg map[string]any
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("read error: %v", err)
				break
			}
			action, _ := msg["action"].(string)

			switch action {
			case "send_group_chat":
				text, _ := msg["message"].(string)
				payload := map[string]any{
					"from":     userID,
					"username": username,
					"message":  text,
					"at":       time.Now().Format(time.RFC3339),
				}
				go l.Store().SaveGroupChat(context.Background(), payload)
				bc.BroadcastToAll(map[string]any{
					"event":   "group_chat",
					"payload": payload,
				})

			case "create_challenge":
				message, _ := msg["message"].(string)
				player := models.Player{ID: userID, Username: username}
				roomID, err := l.CreateChallenge(context.Background(), player, message)
				if err != nil {
					conn.WriteJSON(map[string]any{"error": err.Error()})
					continue
				}
				bc.RegisterConn(roomID, conn)
				conn.WriteJSON(map[string]any{
					"event":   "challenge_created_ack",
					"payload": map[string]any{"room_id": roomID},
				})

			case "accept_challenge":
				roomID, _ := msg["room"].(string)
				player := models.Player{ID: userID, Username: username}

				// Register this connection also under the new game room
				bc.RegisterConn(roomID, conn)

				// Run accept logic
				if err := l.AcceptChallenge(context.Background(), roomID, player); err != nil {
					conn.WriteJSON(map[string]any{"error": err.Error()})
				} else {
					conn.WriteJSON(map[string]any{
						"event": "challenge_joined",
						"payload": map[string]any{
							"room_id": roomID,
							"user":    username,
						},
					})
				}

			case "make_move":
				moveRoomID, _ := msg["room"].(string)
				if moveRoomID == "" {
					moveRoomID = roomID
				}

				idxFloat, _ := msg["index"].(float64)
				idx := int(idxFloat)
				elapsedMs, _ := msg["elapsed_ms"].(float64)
				elapsed := time.Duration(int64(elapsedMs)) * time.Millisecond
				usedGrace, _ := msg["used_grace"].(bool)

				rm, ok := l.GetRoom(moveRoomID)
				if !ok {
					conn.WriteJSON(map[string]any{"error": "room not found"})
					continue
				}

				_ = rm.MakeMove(context.Background(), userID, idx, elapsed, usedGrace)

			case "send_match_chat":
				text, _ := msg["message"].(string)
				if text == "" {
					continue
				}
				payload := map[string]any{
					"from":    userID,
					"message": text,
					"room_id": roomID,
					"at":      time.Now().Format(time.RFC3339),
				}
				go l.Store().SaveMatchChat(context.Background(), roomID, payload)
				bc.BroadcastToRoom(context.Background(), roomID, "match_chat", payload)

			case "get_replay":
				replayRoomID, _ := msg["room"].(string)
				if replayRoomID == "" {
					conn.WriteJSON(map[string]any{"error": "room required"})
					continue
				}

				if l.Store() == nil {
					conn.WriteJSON(map[string]any{"error": "no store configured"})
					continue
				}

				// ✅ Load the board (string key, not list)
				board, err := l.Store().LoadBoard(context.Background(), replayRoomID)
				if err != nil || board == nil {
					conn.WriteJSON(map[string]any{"error": "replay not found"})
					continue
				}

				if len(board.Moves) == 0 {
					conn.WriteJSON(map[string]any{"error": "no moves recorded"})
					continue
				}

				// ✅ Send the move sequence
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
