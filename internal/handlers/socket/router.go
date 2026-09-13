package socket

import (
	"encoding/json"
	"encrypted-db/config"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SocketRouter 

// ServeWSGin wraps ServeWS for use with Gin
func (h *WebSocketHandler) ServeWSGin(c *gin.Context) {
	h.ServeWS(c.Writer, c.Request)
}

// ServeWS handles WebSocket connections
func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer func() {
		// Ensure connection is closed and client is removed on exit
		conn.Close()
		h.Mu.Lock()
		delete(h.Clients, conn)
		h.Mu.Unlock()
		log.Println("WebSocket client disconnected and removed")
	}()

	// Register the new client
	h.Mu.Lock()
	h.Clients[conn] = true
	h.Mu.Unlock()
	log.Println("New WebSocket client connected")

	// Listen for RabbitMQ messages in separate goroutine
	go h.listenRabbitMQ(config.Config.RabbitMQ.Exchanges.Currency, h.handleCurrencyUpdates)

	// Keep WebSocket connection alive and handle client messages
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket client disconnected: %v", err)
			h.Mu.Lock()
			delete(h.Clients, conn)
			h.Mu.Unlock()
			break
		}

		// Process client message (expecting a JSON with "event")
		var event map[string]interface{}
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		// Check for "ping" event
		if evt, ok := event["event"].(string); ok && evt == "ping" {
			log.Println("Received ping from client")

			// Handle ping event by responding with a pong and timestamp
			timezone := "UTC"
			if tz, ok := event["timezone"].(string); ok {
				timezone = tz
			}
			h.handleClientPing(conn, messageType, timezone)
		} else {
			log.Printf("Unhandled event type: %v", event["event"])
		}
	}
}
