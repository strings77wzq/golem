package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	Rate           float64
	Burst          int
	Enabled        bool
	TrustedProxies []string // proxy IPs allowed to set X-Forwarded-For (empty = trust none)
}

// rateLimitStore keeps one official token-bucket limiter per client IP.
// The bucket semantics (burst refill, retry-after computation) are provided
// by golang.org/x/time/rate; the store only owns the per-IP map, its idle
// cleanup, and the last-activity bookkeeping that drives it.
type rateLimitStore struct {
	limiters       map[string]*rate.Limiter
	lastSeen       map[string]time.Time
	mu             sync.Mutex
	rate           rate.Limit
	burst          int
	trustedProxies []string
	done           chan struct{}
}

func newRateLimitStore(r float64, burst int, trustedProxies []string) *rateLimitStore {
	store := &rateLimitStore{
		limiters:       make(map[string]*rate.Limiter),
		lastSeen:       make(map[string]time.Time),
		rate:           rate.Limit(r),
		burst:          burst,
		trustedProxies: trustedProxies,
		done:           make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (rl *rateLimitStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, last := range rl.lastSeen {
				if now.Sub(last) > 10*time.Minute {
					delete(rl.limiters, ip)
					delete(rl.lastSeen, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

func (rl *rateLimitStore) Close() {
	close(rl.done)
}

// allow consumes one token for the IP's bucket. It returns true when the
// request is admitted and the wait duration when it must be retried.
func (rl *rateLimitStore) allow(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	lim, exists := rl.limiters[ip]
	if !exists {
		lim = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = lim
	}

	// Reserve() always succeeds for n <= burst by reserving future tokens;
	// a non-zero Delay means the bucket is empty and the request must wait.
	// A rejected request must not consume quota — Cancel returns the
	// reservation so the bucket recovers at the same pace as a no-op.
	res := lim.Reserve()
	if res.Delay() == 0 {
		// Only admitted requests refresh the idle timer, so a flood of
		// rejected requests cannot pin an entry in the map forever.
		rl.lastSeen[ip] = time.Now()
		return true, 0
	}
	res.Cancel()
	return false, res.Delay()
}

// RateLimitMiddleware returns HTTP middleware that rate-limits by client IP.
// For server shutdown cleanup, use RateLimitMiddlewareWithCleanup instead.
func RateLimitMiddleware(cfg RateLimitConfig) func(http.Handler) http.Handler {
	var store *rateLimitStore
	if cfg.Enabled {
		store = newRateLimitStore(cfg.Rate, cfg.Burst, cfg.TrustedProxies)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r, cfg.TrustedProxies)
			allowed, retryAfter := store.allow(clientIP)

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"}) //nolint:errcheck
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddlewareWithCleanup returns HTTP middleware plus a cleanup
// function that stops the background goroutine. Call the cleanup function
// during server shutdown to avoid leaking the cleanup goroutine.
func RateLimitMiddlewareWithCleanup(cfg RateLimitConfig) (func(http.Handler) http.Handler, func()) {
	var store *rateLimitStore
	if cfg.Enabled {
		store = newRateLimitStore(cfg.Rate, cfg.Burst, cfg.TrustedProxies)
	}

	cleanup := func() {
		if store != nil {
			store.Close()
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r, cfg.TrustedProxies)
			allowed, retryAfter := store.allow(clientIP)

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"}) //nolint:errcheck
				return
			}

			next.ServeHTTP(w, r)
		})
	}, cleanup
}
