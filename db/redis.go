package db

import (
	"admin-stats/config"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisDB implements Database for a Redis connection.
// Access Client to pass *redis.Client into services.
type RedisDB struct {
	Client *redis.Client
	cfg    config.RedisConfig
}

func NewRedisDB(cfg config.RedisConfig) *RedisDB {
	return &RedisDB{cfg: cfg}
}

func (r *RedisDB) Connect(ctx context.Context) error {
	r.Client = redis.NewClient(&redis.Options{
		Addr:     r.cfg.Addr,
		Password: r.cfg.Password,
		DB:       r.cfg.DB,
	})
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return err
	}
	return r.applyServerConfig(ctx)
}

// applyServerConfig sets server-level Redis policies so the client controls
// eviction behaviour rather than relying on the server's default configuration.
func (r *RedisDB) applyServerConfig(ctx context.Context) error {
	if err := r.Client.ConfigSet(ctx, "maxmemory-policy", "allkeys-lru").Err(); err != nil {
		return fmt.Errorf("set maxmemory-policy: %w", err)
	}
	if r.cfg.MaxMemory != "" {
		if err := r.Client.ConfigSet(ctx, "maxmemory", r.cfg.MaxMemory).Err(); err != nil {
			return fmt.Errorf("set maxmemory: %w", err)
		}
	}
	return nil
}

func (r *RedisDB) Disconnect(ctx context.Context) error {
	return r.Client.Close()
}
