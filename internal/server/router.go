// Package server builds the Gin router and runs the HTTP server with graceful
// shutdown.
package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/handler"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/middleware"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/redis"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/session"
)

// Deps are everything the router needs to wire routes and middleware.
type Deps struct {
	Config   *config.Config
	Log      *slog.Logger
	Redis    *redis.Client
	Sessions session.Store
	Handlers *handler.Handlers
}

// NewRouter builds the *gin.Engine with the global middleware chain and routes.
func NewRouter(d Deps) *gin.Engine {
	if d.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// New() not Default() so we control the middleware order explicitly.
	r := gin.New()

	// Global chain — order matters: request ID first (so logs/panics have it),
	// then recovery, logging, CORS, and rate limiting.
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(d.Log),
		middleware.Logger(d.Log),
		middleware.CORS(d.Config.CORS),
		middleware.RateLimit(d.Redis, d.Config.RateLimit),
	)

	// Health probes: no auth, no rate limit concerns (they're above, but
	// orchestrators call these often — acceptable here for simplicity).
	r.GET("/healthz", d.Handlers.Health.Live)
	r.GET("/readyz", d.Handlers.Health.Ready)

	api := r.Group("/api/v1")

	// Public auth routes.
	auth := api.Group("/auth")
	{
		auth.POST("/register", d.Handlers.Auth.Register)
		auth.POST("/login", d.Handlers.Auth.Login)
	}

	// Authenticated routes: require a valid session.
	authed := api.Group("")
	authed.Use(middleware.Auth(d.Sessions, d.Config.Session))
	{
		authed.POST("/auth/logout", d.Handlers.Auth.Logout)
		authed.GET("/auth/me", d.Handlers.Auth.Me)
	}

	// Admin routes: require session AND the "admin" role. Example placeholder.
	admin := api.Group("/admin")
	admin.Use(
		middleware.Auth(d.Sessions, d.Config.Session),
		middleware.RequireRole("admin"),
	)
	{
		admin.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "pong from admin"})
		})
	}

	return r
}
