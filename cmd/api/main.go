package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/database"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/handler"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/migrate"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/redis"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/repository"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/server"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/service"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/session"
	"github.com/Amirreza-Zeraati/go-boilerplate/migrations"
	"github.com/Amirreza-Zeraati/go-boilerplate/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Config.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 2. Logger.
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("starting app", "env", cfg.App.Env, "port", cfg.App.Port)

	// 3. Database + wait for readiness.
	db, err := database.New(cfg.Database, cfg.App.IsProduction())
	if err != nil {
		return err
	}
	defer db.Close()

	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.WaitForConnection(startupCtx, log); err != nil {
		return err
	}
	log.Info("database connected")

	// 4. Migrations — bring schema to newest version on startup (config-gated).
	if cfg.Database.AutoMigrate {
		if err := migrate.Up(migrations.FS, cfg.Database.URL(), log); err != nil {
			return err
		}
	}

	// 5. Redis.
	rdb := redis.New(cfg.Redis)
	defer rdb.Close()
	if err := rdb.Ping(startupCtx); err != nil {
		return err
	}
	log.Info("redis connected")

	// 6. Wire layers: repositories -> services -> session store -> handlers.
	repos := &repository.Repositories{
		User: repository.NewUserRepository(db),
	}
	services := service.New(repos)
	sessions := session.NewRedisStore(rdb, cfg.Session.TTL)

	handlers := &handler.Handlers{
		Auth:   handler.NewAuthHandler(services.Auth, sessions, cfg.Session),
		Health: handler.NewHealthHandler(db, rdb),
	}

	// 7. Router + server.
	router := server.NewRouter(server.Deps{
		Config:   cfg,
		Log:      log,
		Redis:    rdb,
		Sessions: sessions,
		Handlers: handlers,
	})
	srv := server.New(cfg, router)

	// 8. Start in a goroutine so main can wait for a shutdown signal.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start(log)
	}()

	// 9. Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Info("shutdown complete")
	return nil
}
