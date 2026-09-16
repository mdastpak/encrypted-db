package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"encrypted-db/config"

	"github.com/streadway/amqp"
)

type RabbitMQService struct {
	conn        *amqp.Connection
	connMu      sync.RWMutex
	channel     *amqp.Channel
	chMu        sync.RWMutex
	closed      atomic.Bool
	reconnectWg sync.WaitGroup
	notifyClose chan *amqp.Error
	url         string
	exchanges   []string
}

func NewRabbitMQService(exchanges ...string) (*RabbitMQService, error) {
	url := config.GetRabbitMQURL()
	r := &RabbitMQService{
		url:         url,
		exchanges:   exchanges,
		notifyClose: make(chan *amqp.Error, 1),
	}

	if err := r.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	r.reconnectWg.Add(1)
	go r.reconnectLoop()

	return r, nil
}

func (r *RabbitMQService) connect() error {
	r.connMu.Lock()
	defer r.connMu.Unlock()

	conn, err := amqp.DialConfig(r.url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	r.conn = conn
	r.channel = ch
	r.closed.Store(false)

	r.notifyClose = conn.NotifyClose(make(chan *amqp.Error, 1))

	for _, exchange := range r.exchanges {
		if err := r.declareExchange(ch, exchange); err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
		}
	}

	log.Println("RabbitMQ connection and channel established with publisher confirms")
	return nil
}

func (r *RabbitMQService) declareExchange(ch *amqp.Channel, exchangeName string) error {
	return ch.ExchangeDeclare(
		exchangeName,
		"fanout",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
}

func (r *RabbitMQService) GetChannel() (*amqp.Channel, error) {
	r.chMu.RLock()
	ch := r.channel
	r.chMu.RUnlock()

	if ch != nil {
		return ch, nil
	}

	return r.recreateChannel()
}

func (r *RabbitMQService) recreateChannel() (*amqp.Channel, error) {
	r.chMu.Lock()
	defer r.chMu.Unlock()

	if r.channel != nil {
		return r.channel, nil
	}

	r.connMu.RLock()
	conn := r.conn
	r.connMu.RUnlock()

	if conn == nil || conn.IsClosed() {
		if err := r.connect(); err != nil {
			return nil, err
		}
	} else {
		ch, err := conn.Channel()
		if err != nil {
			return nil, err
		}
		if err := ch.Confirm(false); err != nil {
			ch.Close()
			return nil, err
		}
		for _, exchange := range r.exchanges {
			if err := r.declareExchange(ch, exchange); err != nil {
				ch.Close()
				return nil, err
			}
		}
		r.channel = ch
	}
	return r.channel, nil
}

func (r *RabbitMQService) reconnectLoop() {
	defer r.reconnectWg.Done()

	for {
		select {
		case err := <-r.notifyClose:
			if err != nil {
				log.Printf("RabbitMQ connection closed: %v, reconnecting...", err)
			} else {
				log.Println("RabbitMQ connection closed gracefully")
			}
			if r.closed.Load() {
				return
			}
			r.reconnect()
		case <-time.After(30 * time.Second):
			if r.closed.Load() {
				return
			}
			if err := r.healthCheck(); err != nil {
				log.Printf("RabbitMQ health check failed: %v, reconnecting...", err)
				r.reconnect()
			}
		}
	}
}

func (r *RabbitMQService) healthCheck() error {
	r.chMu.RLock()
	ch := r.channel
	r.chMu.RUnlock()

	if ch == nil {
		return fmt.Errorf("no channel")
	}
	return nil
}

// IsHealthy reports whether the service has an active connection and channel.
// It is safe for concurrent use and does not perform any network I/O.
func (r *RabbitMQService) IsHealthy() bool {
	if r.closed.Load() {
		return false
	}

	r.connMu.RLock()
	conn := r.conn
	r.connMu.RUnlock()
	if conn == nil || conn.IsClosed() {
		return false
	}

	return r.healthCheck() == nil
}

func (r *RabbitMQService) reconnect() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for !r.closed.Load() {
		if err := r.connect(); err != nil {
			log.Printf("RabbitMQ reconnect failed: %v, retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		log.Println("RabbitMQ reconnected successfully")
		return
	}
}

func (r *RabbitMQService) Close() error {
	r.closed.Store(true)

	r.chMu.Lock()
	if r.channel != nil {
		r.channel.Close()
		r.channel = nil
	}
	r.chMu.Unlock()

	r.connMu.Lock()
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	r.connMu.Unlock()

	r.reconnectWg.Wait()
	log.Println("RabbitMQ connection closed gracefully")
	return nil
}

func (r *RabbitMQService) Publish(ctx context.Context, exchange, message string) error {
	if r.closed.Load() {
		return fmt.Errorf("rabbitmq service closed")
	}

	ch, err := r.GetChannel()
	if err != nil {
		return err
	}

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = ch.Publish(
		exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         []byte(message),
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		r.chMu.Lock()
		r.channel = nil
		r.chMu.Unlock()
		return err
	}

	select {
	case confirmed := <-confirms:
		if !confirmed.Ack {
			return fmt.Errorf("message nacked by broker")
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("publish confirm timeout")
	}

	return nil
}

func (r *RabbitMQService) PublishWithRetry(ctx context.Context, exchange, message string, retryCount int) error {
	var lastErr error
	for i := 0; i < retryCount; i++ {
		if err := r.Publish(ctx, exchange, message); err != nil {
			lastErr = err
			log.Printf("Failed to publish message, attempt %d/%d: %v", i+1, retryCount, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to publish message after %d attempts: %w", retryCount, lastErr)
}
