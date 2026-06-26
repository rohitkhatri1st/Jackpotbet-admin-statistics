package db

import (
	"admin-stats/config"
	"context"

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
	return r.Client.Ping(ctx).Err()
}

func (r *RedisDB) Disconnect(ctx context.Context) error {
	return r.Client.Close()
}
