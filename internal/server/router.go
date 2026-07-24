// Package server builds the Gin router and runs the HTTP server with graceful
// shutdown.
package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/handler"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/metrics"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/middleware"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/redis"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/routes"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/session"
)

// Deps are everything the router needs to wire routes and middleware.
type Deps struct {
	Config   *config.Config
	Log      *slog.Logger
	Redis    *redis.Client
	Sessions session.Store
	Handlers *handler.Handlers
	Metrics  *metrics.Metrics
}

// NewRouter builds the *gin.Engine with the global middleware chain and routes.
func NewRouter(d Deps) *gin.Engine {
	if d.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// New() not Default() so we control the middleware order explicitly.
	r := gin.New()

	// Global chain — order matters: request ID first (so logs/panics have it),
	// then recovery, logging, metrics, CORS.
	//
	// Rate limiting is deliberately NOT global: it's applied to /api/v1 below,
	// so health probes and metrics scrapes are never throttled.
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(d.Log),
		middleware.Logger(d.Log),
	)
	if d.Metrics != nil {
		r.Use(middleware.Metrics(d.Metrics))
	}
	r.Use(middleware.CORS(d.Config.CORS))

	// Health probes: no auth, no rate limit.
	r.GET("/healthz", d.Handlers.Health.Live)
	r.GET("/readyz", d.Handlers.Health.Ready)

	// Prometheus scrape endpoint.
	if d.Config.Metrics.Enabled && d.Metrics != nil {
		r.GET(d.Config.Metrics.Path, gin.WrapH(d.Metrics.Handler()))
	}

	api := r.Group("/api/v1")
	api.Use(middleware.RateLimit(d.Redis, d.Config.RateLimit))

	routes.Register(api, routes.Deps{
		Config:   d.Config,
		Handlers: d.Handlers,
		Sessions: d.Sessions,
	})

	return r
}
