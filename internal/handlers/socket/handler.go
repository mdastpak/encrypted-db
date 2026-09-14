package socket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"encrypted-db/config"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	RabbitMQService *rabbitmq.RabbitMQService
	Clients         map[*Client]bool
	Mu              sync.RWMutex
	Upgrader        websocket.Upgrader
	ShutdownCh      chan struct{}
	Wg              sync.WaitGroup
}

type Client struct {
	Conn     *websocket.Conn
	Send     chan []byte
	Timezone string
}

var (
	// Default upgrader with secure origin check
	defaultUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// In production, configure allowed origins from config
			origin := r.Header.Get("Origin")
			allowed := config.Config.WebSocket.AllowedOrigins
			if len(allowed) == 0 {
				// Default: allow same origin only
				return origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host
			}
			for _, o := range allowed {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func NewWebSocketHandler(is *models.InfraServices) *WebSocketHandler {
	h := &WebSocketHandler{
		RabbitMQService: is.RabbitMQ,
		Clients:         make(map[*Client]bool),
		Upgrader:        defaultUpgrader,
		ShutdownCh:      make(chan struct{}),
	}
	return h
}

func (h *WebSocketHandler) ServeWSGin(c *gin.Context) {
	h.ServeWS(c.Writer, c.Request)
}

func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Timezone: "UTC",
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	h.Wg.Add(1)
	go h.writePump(client)

	h.readPump(client)
}

func (h *WebSocketHandler) registerClient(client *Client) {
	h.Mu.Lock()
	h.Clients[client] = true
	h.Mu.Unlock()
	log.Printf("WebSocket client connected. Total: %d", h.clientCount())
}

func (h *WebSocketHandler) unregisterClient(client *Client) {
	h.Mu.Lock()
	delete(h.Clients, client)
	h.Mu.Unlock()
	close(client.Send)
	client.Conn.Close()
	log.Printf("WebSocket client disconnected. Total: %d", h.clientCount())
}

func (h *WebSocketHandler) clientCount() int {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	return len(h.Clients)
}

func (h *WebSocketHandler) readPump(client *Client) {
	defer func() {
		h.unregisterClient(client)
		h.Wg.Done()
	}()

	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-h.ShutdownCh:
			return
		default:
			messageType, message, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}

			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}

			h.handleClientMessage(client, message)
		}
	}
}

func (h *WebSocketHandler) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		h.Wg.Done()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-h.ShutdownCh:
			return
		}
	}
}

func (h *WebSocketHandler) handleClientMessage(client *Client, message []byte) {
	var event map[string]interface{}
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid message format: %v", err)
		return
	}

	evt, ok := event["event"].(string)
	if !ok {
		return
	}

	switch evt {
	case "ping":
		if tz, ok := event["timezone"].(string); ok {
			client.Timezone = tz
		}
		h.handleClientPing(client)

	case "subscribe":
		// Future: handle channel subscriptions
		log.Printf("Client subscribed to: %v", event["channels"])

	default:
		log.Printf("Unhandled event type: %s", evt)
	}
}

func (h *WebSocketHandler) handleClientPing(client *Client) {
	loc, err := time.LoadLocation(client.Timezone)
	if err != nil {
		loc = time.UTC
	}
	serverTime := time.Now().In(loc).Format(time.RFC3339)

	pongMessage := map[string]interface{}{
		"event":     "pong",
		"timestamp": serverTime,
	}

	data, err := json.Marshal(pongMessage)
	if err != nil {
		log.Printf("Error marshaling pong: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Printf("Client send buffer full, dropping ping")
	}
}

func (h *WebSocketHandler) Broadcast(message []byte) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	for client := range h.Clients {
		select {
		case client.Send <- message:
		default:
			log.Printf("Client send buffer full, skipping broadcast")
		}
	}
}

func (h *WebSocketHandler) StartConsumer(ctx context.Context) error {
	h.Wg.Add(1)
	go func() {
		defer h.Wg.Done()
		h.consumeRabbitMQ(ctx)
	}()
	return nil
}

func (h *WebSocketHandler) consumeRabbitMQ(ctx context.Context) {
	exchangeName := config.Config.RabbitMQ.Exchanges.Currency
	queueName := "ws_currency_updates"

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.ShutdownCh:
			return
		default:
			err := h.consumeOnce(ctx, exchangeName, queueName)
			if err != nil {
				log.Printf("RabbitMQ consumer error: %v, reconnecting in 5s...", err)
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return
				case <-h.ShutdownCh:
					return
				}
			}
		}
	}
}

func (h *WebSocketHandler) consumeOnce(ctx context.Context, exchangeName, queueName string) error {
	ch, err := h.RabbitMQService.GetChannel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		queueName,
		"",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("WebSocket consumer started on queue: %s", queueName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-h.ShutdownCh:
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil // channel closed
			}
			h.handleCurrencyUpdate(msg.Body)
			msg.Ack(false)
		}
	}
}

func (h *WebSocketHandler) handleCurrencyUpdate(body []byte) {
	var update map[string]interface{}
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Error unmarshaling currency update: %v", err)
		return
	}

	// Add server timestamp
	update["server_time"] = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling broadcast: %v", err)
		return
	}

	h.Broadcast(data)
}

func (h *WebSocketHandler) Shutdown(ctx context.Context) error {
	close(h.ShutdownCh)
	h.Mu.Lock()
	for client := range h.Clients {
		client.Conn.Close()
	}
	h.Mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.Wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
