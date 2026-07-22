package pubsub

import (
	"context"
	"sync"

	"posts-service/internal/model"
)

const subscriberBuffer = 32

type CommentBus struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[string]map[uint64]chan *model.Comment
}

func NewCommentBus() *CommentBus {
	return &CommentBus{subs: make(map[string]map[uint64]chan *model.Comment)}
}

func (b *CommentBus) Subscribe(ctx context.Context, postID string) <-chan *model.Comment {
	ch := make(chan *model.Comment, subscriberBuffer)

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	if b.subs[postID] == nil {
		b.subs[postID] = make(map[uint64]chan *model.Comment)
	}
	b.subs[postID][id] = ch
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subs[postID], id)
		if len(b.subs[postID]) == 0 {
			delete(b.subs, postID)
		}
		b.mu.Unlock()
		close(ch)
	}()

	return ch
}

func (b *CommentBus) Publish(comment *model.Comment) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subs[comment.PostID] {
		select {
		case ch <- comment:
		default:
		}
	}
}

func (b *CommentBus) SubscribersCount(postID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[postID])
}
