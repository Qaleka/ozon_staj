package pubsub

import (
	"context"
	"testing"
	"time"

	"posts-service/internal/model"
)

func TestSubscribePublish(t *testing.T) {
	bus := NewCommentBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1 := bus.Subscribe(ctx, "post1")
	ch2 := bus.Subscribe(ctx, "post1")
	chOther := bus.Subscribe(ctx, "post2")

	comment := &model.Comment{ID: "c1", PostID: "post1", Text: "hi"}
	bus.Publish(comment)

	for _, ch := range []<-chan *model.Comment{ch1, ch2} {
		select {
		case got := <-ch:
			if got.ID != "c1" {
				t.Errorf("got comment %q, want c1", got.ID)
			}
		case <-time.After(time.Second):
			t.Fatal("subscriber did not receive comment")
		}
	}

	select {
	case c := <-chOther:
		t.Errorf("post2 subscriber must not receive %q", c.ID)
	default:
	}
}

func TestUnsubscribeOnContextCancel(t *testing.T) {
	bus := NewCommentBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.Subscribe(ctx, "post1")
	if n := bus.SubscribersCount("post1"); n != 1 {
		t.Fatalf("subscribers = %d, want 1", n)
	}

	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected closed channel without values")
		}
	case <-time.After(time.Second):
		t.Fatal("channel was not closed after context cancel")
	}

	deadline := time.Now().Add(time.Second)
	for bus.SubscribersCount("post1") != 0 {
		if time.Now().After(deadline) {
			t.Fatal("subscription was not removed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSlowSubscriberDoesNotBlock(t *testing.T) {
	bus := NewCommentBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = bus.Subscribe(ctx, "post1")

	done := make(chan struct{})
	go func() {
		for i := 0; i < subscriberBuffer*2; i++ {
			bus.Publish(&model.Comment{ID: "c", PostID: "post1"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on slow subscriber")
	}
}
