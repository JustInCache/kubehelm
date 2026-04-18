package service

import (
	"encoding/json"
	"log/slog"
	"sync"
)

type StreamHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
	logger  *slog.Logger
}

func NewStreamHub(logger *slog.Logger) *StreamHub {
	return &StreamHub{
		clients: make(map[chan []byte]struct{}),
		logger:  logger,
	}
}

func (h *StreamHub) Register() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *StreamHub) Unregister(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *StreamHub) Broadcast(payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		h.logger.Warn("stream marshal failed", slog.String("error", err.Error()))
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- body:
		default:
		}
	}
}
