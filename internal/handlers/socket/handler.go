package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"encrypted-db/config"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	RabbitMQService *rabbitmq.RabbitMQService
	Clients         map[*websocket.Conn]bool
	Mu              sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// NewWebSocketHandler initializes a WebSocketHandler with RabbitMQ service
// func NewWebSocketHandler(rabbitMQService *rabbitmq.RabbitMQService) *WebSocketHandler {
func NewWebSocketHandler(is *models.InfraServices) *WebSocketHandler {
	return &WebSocketHandler{
		RabbitMQService: is.RabbitMQ,
		Clients:         make(map[*websocket.Conn]bool),
	}
}

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

func (h *WebSocketHandler) listenRabbitMQ(queueName string, handlerFunc func([]byte)) {
	// Declare the queue if it does not exist
	queue, err := h.RabbitMQService.Channel.QueueDeclare(
		queueName, // Queue name
		true,      // Durable
		false,     // Auto-delete
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		log.Printf("Error declaring RabbitMQ queue %s: %v", queueName, err)
		return
	}

	// Bind the queue to the exchange to receive messages
	err = h.RabbitMQService.Channel.QueueBind(
		queue.Name,          // Queue name
		"",                  // Routing key (empty for fanout)
		"currency_exchange", // Exchange name
		false,               // No-wait
		nil,                 // Arguments
	)
	if err != nil {
		log.Printf("Error binding queue to exchange %s: %v", "currency_exchange", err)
		return
	}

	log.Printf("Listening for RabbitMQ messages on queue %s", queueName)

	// Start consuming messages from the declared queue
	msgs, err := h.RabbitMQService.Channel.Consume(
		queue.Name, // Queue name
		"",         // Consumer tag
		true,       // Auto-ack
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		log.Printf("Error consuming RabbitMQ messages from queue %s: %v", queueName, err)
		return
	}

	log.Printf("Consuming RabbitMQ messages from queue %s", queueName)

	// Process each message using the provided handler function
	for msg := range msgs {
		handlerFunc(msg.Body)
	}
}
