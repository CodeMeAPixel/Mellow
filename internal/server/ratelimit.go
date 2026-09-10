package server

import (
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

func rateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	buckets := map[string]*bucket{}

	go func() {
		for range time.Tick(window) {
			mu.Lock()
			now := time.Now()
			for k, b := range buckets {
				if now.After(b.resetAt) {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr

			mu.Lock()
			b, ok := buckets[key]
			now := time.Now()
			if !ok || now.After(b.resetAt) {
				b = &bucket{resetAt: now.Add(window)}
				buckets[key] = b
			}
			b.count++
			over := b.count > limit
			retry := time.Until(b.resetAt).Seconds()
			mu.Unlock()

			if over {
				w.Header().Set("Retry-After", itoa(int(retry)+1))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
