package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

// RequestID assigns each request a correlation ID (reusing an inbound
// X-Request-ID if present) and echoes it back in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(ctxRequestID, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// Logger logs one structured line per request after it completes.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", RequestID(),
		}
		switch {
		case c.Writer.Status() >= 500:
			log.Error("request", attrs...)
		case c.Writer.Status() >= 400:
			log.Warn("request", attrs...)
		default:
			log.Info("request", attrs...)
		}
	}
}

// Recovery converts panics into a clean JSON 500 (instead of gin's default
// plaintext) and logs the panic with its request ID.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					"error", r,
					"path", c.Request.URL.Path,
					"request_id", RequestID(),
				)
				response.AbortError(c, http.StatusInternalServerError, "internal server error")
			}
		}()
		c.Next()
	}
}
