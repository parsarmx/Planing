package realtime

import "sync"

type Hub struct {
	mu        sync.Mutex
	listeners map[string]map[chan struct{}]string
	done      chan struct{}
	stopped   bool
}

func New() *Hub {
	return &Hub{listeners: map[string]map[chan struct{}]string{}, done: make(chan struct{})}
}
func (h *Hub) Subscribe(room, user string) (chan struct{}, func()) {
	h.mu.Lock()
	count := 0
	for _, listeners := range h.listeners {
		for _, id := range listeners {
			if id == user {
				count++
			}
		}
	}
	if h.stopped || count >= 6 {
		h.mu.Unlock()
		return nil, func() {}
	}
	users := map[string]bool{}
	for _, id := range h.listeners[room] {
		users[id] = true
	}
	if !users[user] && len(users) >= 20 {
		h.mu.Unlock()
		return nil, func() {}
	}
	if h.listeners[room] == nil {
		h.listeners[room] = map[chan struct{}]string{}
	}
	ch := make(chan struct{}, 1)
	h.listeners[room][ch] = user
	h.mu.Unlock()
	h.Publish(room)
	return ch, func() {
		h.mu.Lock()
		delete(h.listeners[room], ch)
		if len(h.listeners[room]) == 0 {
			delete(h.listeners, room)
		}
		h.mu.Unlock()
		h.Publish(room)
	}
}
func (h *Hub) Publish(room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.listeners[room] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (h *Hub) Online(room, user string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range h.listeners[room] {
		if id == user {
			return true
		}
	}
	return false
}

func (h *Hub) Count(room string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	users := map[string]bool{}
	for _, id := range h.listeners[room] {
		users[id] = true
	}
	return len(users)
}

func (h *Hub) Done() <-chan struct{} { return h.done }
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.stopped {
		h.stopped = true
		close(h.done)
	}
}
