package graph

import (
	"context"
	"encrypted-db/graph/model"
	"time"
)

func (r *Resolver) AddMessage(ctx context.Context, content string, sender string) (*model.Message, error) {
	// Create a new message
	message := &model.Message{
		ID:        generateID(),
		Content:   content,
		Sender:    sender,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Send message to subscribers
	r.mu.Lock()
	for _, subscriber := range r.MessageSubscribers {
		subscriber <- message
	}
	r.mu.Unlock()

	return message, nil
}
