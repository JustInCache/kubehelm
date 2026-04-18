package middleware

import (
	"net/http"
	"sync"
	"time"
)

// Cheap token-bucket limiter for expensive endpoints per remote IP.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*bucket
	rps     float64
	burst   float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewExpensiveRateLimiter(rps, burst int) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		entries: make(map[string]*bucket),
		rps:     float64(rps),
		burst:   float64(burst),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl.allow(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Too many expensive requests"}`))
		})
	}
}

func (r *rateLimiter) allow(key string) bool {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.entries[key]
	if !ok {
		r.entries[key] = &bucket{tokens: r.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * r.rps
	if b.tokens > r.burst {
		b.tokens = r.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
