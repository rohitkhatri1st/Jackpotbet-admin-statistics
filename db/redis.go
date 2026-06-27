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
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return err
	}
	return r.applyServerConfig(ctx)
}

// applyServerConfig sets Redis server-level policies at connect time.
//
// allkeys-lru is always enforced: when Redis runs out of memory it evicts the
// least-recently-used key automatically, which is the correct behaviour for a
// stat result cache — stale entries are dropped under pressure rather than
// blocking new writes.
//
// maxmemory is only applied when explicitly configured (e.g. "256mb"). Leaving
// it empty delegates memory management to the operator's Redis configuration.
//
// NOTE: This caching setup pairs with simple TTL-based key caching in the
// service layer. A more scalable approach would use a materialized view (a
// pre-computed `daily_stats` MongoDB collection filled by a nightly cron) so
// that Redis caches a tiny read-through layer instead of raw aggregation results.
// See TransactionService for details on why that is not implemented here.
func (r *RedisDB) applyServerConfig(ctx context.Context) error {
	if err := r.Client.ConfigSet(ctx, "maxmemory-policy", "allkeys-lru").Err(); err != nil {
		return err
	}
	if r.cfg.MaxMemory != "" {
		return r.Client.ConfigSet(ctx, "maxmemory", r.cfg.MaxMemory).Err()
	}
	return nil
}

func (r *RedisDB) Disconnect(ctx context.Context) error {
	return r.Client.Close()
}
