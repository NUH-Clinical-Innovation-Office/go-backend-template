package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit returns a middleware that applies a per-client-IP token-bucket
// rate limit. The first request from a new IP gets a fresh bucket sized to
// burst; subsequent requests are limited to rps tokens per second.
//
// Stale entries are removed on access (no background goroutine needed):
// each lookup checks the last-seen timestamp and evicts buckets idle for
// more than idleTimeout. This keeps the map bounded without a janitor.
func RateLimit(rps int, burst int, idleTimeout time.Duration) func(http.Handler) http.Handler {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = rps
	}
	if idleTimeout <= 0 {
		idleTimeout = 10 * time.Minute
	}

	type entry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]*entry)
	)

	evict := func() {
		cutoff := time.Now().Add(-idleTimeout)
		for ip, e := range buckets {
			if e.lastSeen.Before(cutoff) {
				delete(buckets, ip)
			}
		}
	}

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		if e, ok := buckets[ip]; ok {
			e.lastSeen = time.Now()
			return e.limiter
		}
		lim := rate.NewLimiter(rate.Limit(rps), burst)
		buckets[ip] = &entry{limiter: lim, lastSeen: time.Now()}
		return lim
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _ := r.Context().Value(ClientIPKey).(string)
			if ip == "" {
				ip = GetRealIP(r)
			}

			// Opportunistic eviction. Cheap and bounds the map.
			mu.Lock()
			evictLocked := len(buckets) > 256
			mu.Unlock()
			if evictLocked {
				mu.Lock()
				evict()
				mu.Unlock()
			}

			if !getLimiter(ip).Allow() {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"detail":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
