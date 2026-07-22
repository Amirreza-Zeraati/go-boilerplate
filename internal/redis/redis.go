// Package redis owns the Redis client lifecycle. Other packages depend on this
// wrapper rather than importing go-redis directly.
package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
)

// Nil is returned by the client when a key does not exist. Re-exported so
// callers can check for it without importing go-redis directly.
var Nil = goredis.Nil

// Client wraps the go-redis client so swapping libraries touches one file.
type Client struct {
	*goredis.Client
}

// New builds a Redis client. It does not ping; call Ping separately so the
// caller controls the timeout.
func New(cfg config.Redis) *Client {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Client{rdb}
}

// Ping verifies connectivity, honoring the context deadline. Use in readiness
// checks.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close releases the connection pool.
func (c *Client) Close() error {
	return c.Client.Close()
}
