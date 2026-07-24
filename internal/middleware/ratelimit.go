package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/apperr"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/redis"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/response"
)

// RateLimit applies a fixed-window limit per client IP, backed by Redis so the
// limit is shared across all app instances. It uses a single pipelined
// INCR + EXPIRE (2 round-trips collapsed into 1).
func RateLimit(rdb *redis.Client, cfg config.RateLimit) gin.HandlerFunc {
	windowSecs := strconv.Itoa(int(cfg.Window.Seconds()))

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		key := "ratelimit:" + c.ClientIP()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		pipe := rdb.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, cfg.Window)
		if _, err := pipe.Exec(ctx); err != nil {
			// Fail open: if Redis is briefly unavailable, don't lock users out.
			c.Next()
			return
		}

		count := incr.Val()
		remaining := int64(cfg.Requests) - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if count > int64(cfg.Requests) {
			c.Header("Retry-After", windowSecs)
			response.AbortFail(c, apperr.RateLimited("rate limit exceeded"))
			return
		}
		c.Next()
	}
}
