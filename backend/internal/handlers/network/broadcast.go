package network

import (
	"encoding/json"
	"log"
)

// BroadcastEvent sends a structured event to all connected WebSocket clients.
// It wraps the event type + payload into a JSON object and pushes it to the broadcast channel.
func BroadcastEvent(eventType string, payload interface{}) {
	data := map[string]interface{}{
		"type": eventType,
		"data": payload,
	}
	bytes, _ := json.Marshal(data)

	select {
	case broadcast <- bytes:
		log.Printf("Broadcasted event: %s", eventType)
	default:
		log.Println("Broadcast channel full — event dropped:", eventType)
	}
}

func BroadcastGameEvent(gameID string, eventType string, payload interface{}) {
	data := map[string]interface{}{
		"type": eventType,
		"data": payload,
	}
	bytes, _ := json.Marshal(data)
	go broadcastToGame(gameID, bytes)
}
