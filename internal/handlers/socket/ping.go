package socket

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// handleClientPing responds to ping events with a pong message containing server timestamp
func (h *WebSocketHandler) handleClientPing(conn *websocket.Conn, messageType int, timezone string) {
	// Convert current time to specified timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Invalid timezone, defaulting to UTC: %v", err)
		loc = time.UTC
	}
	serverTime := time.Now().In(loc).Format(time.RFC3339)

	// Respond with pong message
	pongMessage := fmt.Sprintf(`{"event": "pong", "timestamp": "%s"}`, serverTime)
	if err := conn.WriteMessage(messageType, []byte(pongMessage)); err != nil {
		log.Printf("Error sending pong message: %v", err)
	}
}
