package socket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCurrencyUpdates_BroadcastsWithServerTime(t *testing.T) {
	h := newTestHandler()
	client := newTestClient()
	h.Mu.Lock()
	h.Clients[client] = true
	h.Mu.Unlock()

	body := []byte(`{"hk":"abc","action":"update"}`)
	h.handleCurrencyUpdates(body)

	select {
	case got := <-client.Send:
		var msg map[string]interface{}
		require.NoError(t, json.Unmarshal(got, &msg))
		assert.Equal(t, "abc", msg["hk"])
		assert.Equal(t, "update", msg["action"])
		assert.NotEmpty(t, msg["server_time"])
	default:
		t.Fatal("expected the currency update to be broadcast to the client")
	}
}

func TestHandleCurrencyUpdates_InvalidJSONDoesNotBroadcast(t *testing.T) {
	h := newTestHandler()
	client := newTestClient()
	h.Mu.Lock()
	h.Clients[client] = true
	h.Mu.Unlock()

	assert.NotPanics(t, func() {
		h.handleCurrencyUpdates([]byte("not-json"))
	})
	assert.Empty(t, client.Send)
}

func TestHandleCurrencyUpdate_BroadcastsWithServerTime(t *testing.T) {
	h := newTestHandler()
	client := newTestClient()
	h.Mu.Lock()
	h.Clients[client] = true
	h.Mu.Unlock()

	h.handleCurrencyUpdate([]byte(`{"hk":"xyz"}`))

	select {
	case got := <-client.Send:
		var msg map[string]interface{}
		require.NoError(t, json.Unmarshal(got, &msg))
		assert.Equal(t, "xyz", msg["hk"])
		assert.NotEmpty(t, msg["server_time"])
	default:
		t.Fatal("expected the currency update to be broadcast to the client")
	}
}

// consumeOnce fails fast (no live broker in this environment): GetChannel
// tries to recreate a channel, which requires dialing RabbitMQ, and a
// zero-value RabbitMQService has an empty URL that amqp rejects immediately.
func TestConsumeOnce_FailsFastWithoutBroker(t *testing.T) {
	h := &WebSocketHandler{RabbitMQService: &rabbitmq.RabbitMQService{}}

	err := h.consumeOnce(context.Background(), "some-exchange", "some-queue")
	assert.Error(t, err)
}

func TestStartConsumer_ShutdownStopsConsumerLoopPromptly(t *testing.T) {
	h := NewWebSocketHandler(&models.InfraServices{RabbitMQ: &rabbitmq.RabbitMQService{}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, h.StartConsumer(ctx))

	done := make(chan struct{})
	go func() {
		h.Shutdown(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("expected Shutdown to stop the consumer loop promptly")
	}
}
