package api

import "sync"

// Hub broadcasts messages to every subscribed client (e.g. each open browser tab).
type Hub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[chan []byte]struct{})}
}

// Subscribe registers a new client. It returns the channel the client reads
// from, and a function the client must call when it disconnects.
func (h *Hub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 64)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, ch)
			close(ch)
			h.mu.Unlock()
		})
	}
	return ch, unsubscribe
}

// Publish sends msg to every subscriber. A client that is too slow to keep up
// has the message dropped, so one slow browser can never freeze the system.
func (h *Hub) Publish(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default: // subscriber's buffer is full: drop this message for them
		}
	}
}

// Count returns how many clients are subscribed.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
