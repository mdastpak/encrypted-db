package rabbitmq

import (
	"fmt"
	"log"
	"sync"
	"time"

	"encrypted-db/config"

	"github.com/streadway/amqp"
)

type RabbitMQService struct {
	Connection    *amqp.Connection
	Channel       *amqp.Channel
	isChannelOpen bool
	mu            sync.Mutex
}

// NewRabbitMQService initializes a new RabbitMQ connection and channel
func NewRabbitMQService() *RabbitMQService {
	r := &RabbitMQService{}
	if err := r.connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	// create the exchanges
	if err := r.createExchange("currency_exchange"); err != nil {
		log.Fatalf("Failed to create exchange: %v", err)
	}

	return r
}

// connect establishes a new connection and channel to RabbitMQ
func (r *RabbitMQService) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize connection
	url := config.GetRabbitMQURL()
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	r.Connection = conn
	r.Channel = ch
	r.isChannelOpen = true
	log.Println("RabbitMQ connection and channel established.")
	return nil
}

// Close closes the RabbitMQ connection and channel
func (r *RabbitMQService) Close() {
	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		}
		r.isChannelOpen = false
	}
	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}
}

func (r *RabbitMQService) createExchange(exchangeName string) error {
	// Declare the exchange if it does not exist
	err := r.Channel.ExchangeDeclare(
		exchangeName, // Exchange name
		"fanout",     // Exchange type (fanout for broadcasting)
		true,         // Durable
		false,        // Auto-deleted when unused
		false,        // Internal
		false,        // No-wait
		nil,          // Arguments
	)
	if err != nil {
		log.Printf("Error declaring RabbitMQ exchange %s: %v", exchangeName, err)
	}
	return err
}

// ensureConnectionAndChannel checks if the connection and channel are open, and recreates them if necessary
func (r *RabbitMQService) ensureConnectionAndChannel() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if connection is closed and attempt to reconnect
	if r.Connection == nil || r.Connection.IsClosed() {
		log.Println("RabbitMQ connection is closed. Reconnecting...")
		if err := r.connect(); err != nil {
			return err
		}
	}

	// Check if channel is open and recreate it if necessary
	if r.Channel == nil || !r.isChannelOpen {
		ch, err := r.Connection.Channel()
		if err != nil {
			log.Printf("Failed to reopen RabbitMQ channel: %v", err)
			r.isChannelOpen = false
			return err
		}
		r.Channel = ch
		r.isChannelOpen = true
	}
	return nil
}

// PublishWithRetry tries to publish a message and retries if it fails
func (r *RabbitMQService) PublishWithRetry(exchange, message string, retryCount int) error {
	for i := 0; i < retryCount; i++ {
		if err := r.Publish(exchange, message); err != nil {
			log.Printf("Failed to publish message, attempt %d/%d: %v", i+1, retryCount, err)
			time.Sleep(500 * time.Millisecond) // Wait before retrying
		} else {
			return nil // Success
		}
	}
	return fmt.Errorf("failed to publish message after %d attempts", retryCount)
}

// Publish sends a message to the specified RabbitMQ exchange
func (r *RabbitMQService) Publish(exchange, message string) error {
	// Ensure connection and channel are open before publishing
	if err := r.ensureConnectionAndChannel(); err != nil {
		return err
	}

	// Publish the message to the exchange
	err := r.Channel.Publish(
		exchange, // Exchange name
		"",       // Routing key (empty for fanout)
		false,    // Mandatory
		false,    // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(message),
		},
	)
	if err != nil {
		log.Printf("Error publishing message to RabbitMQ exchange %s: %v", exchange, err)
		r.isChannelOpen = false // Mark channel as closed if publish fails
	}
	return err
}
