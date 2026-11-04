package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	start time.Time
	count int
}

// RateLimiter: fixed window per menit, per API key client (fallback IP).
// In-memory; untuk multi-instance pindahkan ke Redis.
func RateLimiter(perMinute int) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]*window{}

	go func() {
		for range time.Tick(5 * time.Minute) {
			mu.Lock()
			for k, w := range buckets {
				if time.Since(w.start) > time.Minute {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(x *gin.Context) {
		key := x.GetHeader("X-API-Key")
		if key == "" {
			key = "ip:" + x.ClientIP()
		}
		mu.Lock()
		w := buckets[key]
		now := time.Now()
		if w == nil || now.Sub(w.start) >= time.Minute {
			w = &window{start: now}
			buckets[key] = w
		}
		w.count++
		over := w.count > perMinute
		mu.Unlock()
		if over {
			x.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"success": false, "error": "rate limit exceeded"})
			return
		}
		x.Next()
	}
}
