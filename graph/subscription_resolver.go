package graph

import (
	"context"
	"encrypted-db/graph/model"
	"fmt"
	"time"
)

// MessageAdded returns a channel that streams new messages to subscribers
func (r *Resolver) MessageAdded(ctx context.Context) (<-chan *model.Message, error) {
	id := generateID()
	ch := make(chan *model.Message, 1)

	r.mu.Lock()
	r.MessageSubscribers[id] = ch
	r.mu.Unlock()

	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(r.MessageSubscribers, id)
		r.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
