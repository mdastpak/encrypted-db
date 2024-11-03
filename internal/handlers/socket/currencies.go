package socket

import (
	"log"

	"github.com/gorilla/websocket"
)

// handleCurrencyUpdates processes currency update messages and broadcasts them to WebSocket clients
func (h *WebSocketHandler) handleCurrencyUpdates(message []byte) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	for client := range h.Clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error sending currency update to WebSocket client: %v", err)
			client.Close()
			delete(h.Clients, client)
		}
	}
}
