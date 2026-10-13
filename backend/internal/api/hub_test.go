package api

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSubscriberReceivesPublishedMessage(t *testing.T) {
	h := NewHub()
	ch, unsubscribe := h.Subscribe()
	defer unsubscribe()

	h.Publish([]byte("hello"))

	select {
	case msg := <-ch:
		if string(msg) != "hello" {
			t.Fatalf("got %q, want %q", msg, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("message never arrived")
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	h := NewHub()
	ch, unsubscribe := h.Subscribe()

	unsubscribe()
	unsubscribe() // calling twice must be safe

	if _, open := <-ch; open {
		t.Fatal("channel should be closed after unsubscribe")
	}
	if h.Count() != 0 {
		t.Fatalf("want 0 subscribers, got %d", h.Count())
	}
}

func TestSlowSubscriberDoesNotBlockPublish(t *testing.T) {
	h := NewHub()
	_, unsubscribe := h.Subscribe() // never reads
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ { // far more than the 64-message buffer
			h.Publish([]byte("x"))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}

func TestConcurrentPublishAndSubscribe(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Go(func() {
			ch, unsubscribe := h.Subscribe()
			defer unsubscribe()
			h.Publish([]byte(fmt.Sprintf("msg-%d", i)))
			<-ch
		})
	}

	wg.Wait()
	if h.Count() != 0 {
		t.Fatalf("all clients left, but %d still subscribed", h.Count())
	}
}
