package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/database"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/redis"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	db  *database.DB
	rdb *redis.Client
}

// NewHealthHandler builds the health handler.
func NewHealthHandler(db *database.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

// Live is a liveness probe: is the process up? It checks no dependencies, so a
// slow database never causes the orchestrator to kill a healthy process.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready is a readiness probe: can the app serve traffic? It pings every
// critical dependency and returns 503 if any is down, so load balancers stop
// routing until the app is truly ready.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{"database": "ok", "redis": "ok"}
	ready := true

	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "unavailable"
		ready = false
	}
	if err := h.rdb.Ping(ctx); err != nil {
		checks["redis"] = "unavailable"
		ready = false
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"ready": ready, "checks": checks})
}
