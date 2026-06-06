package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type windowCounter struct {
	mu     sync.Mutex
	hits   []time.Time
	limit  int
	window time.Duration
}

func newWindowCounter(limit int, window time.Duration) *windowCounter {
	return &windowCounter{limit: limit, window: window}
}

func (w *windowCounter) allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-w.window)

	filtered := w.hits[:0]
	for _, t := range w.hits {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	w.hits = filtered

	if len(w.hits) >= w.limit {
		return false
	}
	w.hits = append(w.hits, now)
	return true
}

type RateLimiter struct {
	mu        sync.Mutex
	merchants map[string]*windowCounter
	ips       map[string]*windowCounter

	merchantLimit int
	ipLimit       int
	window        time.Duration
}

func NewRateLimiter(merchantLimit, ipLimit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		merchants:     make(map[string]*windowCounter),
		ips:           make(map[string]*windowCounter),
		merchantLimit: merchantLimit,
		ipLimit:       ipLimit,
		window:        window,
	}
}

func (r *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ipKey := c.IP()
		if !r.allowIP(ipKey) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "rate limit exceeded",
			})
		}

		if apiKey := c.Get(headerMerchantKey); apiKey != "" {
			if !r.allowMerchant(apiKey) {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"success": false,
					"error":   "merchant rate limit exceeded",
				})
			}
		}

		return c.Next()
	}
}

func (r *RateLimiter) allowIP(key string) bool {
	r.mu.Lock()
	counter, ok := r.ips[key]
	if !ok {
		counter = newWindowCounter(r.ipLimit, r.window)
		r.ips[key] = counter
	}
	r.mu.Unlock()
	return counter.allow()
}

func (r *RateLimiter) allowMerchant(key string) bool {
	r.mu.Lock()
	counter, ok := r.merchants[key]
	if !ok {
		counter = newWindowCounter(r.merchantLimit, r.window)
		r.merchants[key] = counter
	}
	r.mu.Unlock()
	return counter.allow()
}
