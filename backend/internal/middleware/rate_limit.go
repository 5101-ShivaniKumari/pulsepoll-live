package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	lastSeen time.Time
	count    int
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int           // Max requests
	window   time.Duration // Time window
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}

	// Background cleanup of stale visitor records
	go func() {
		ticker := time.NewTicker(rl.window * 2)
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		now := time.Now()

		if !exists || now.Sub(v.lastSeen) > rl.window {
			rl.visitors[ip] = &visitor{lastSeen: now, count: 1}
			rl.mu.Unlock()
			c.Next()
			return
		}

		v.count++
		v.lastSeen = now

		if v.count > rl.limit {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please slow down.",
			})
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}
