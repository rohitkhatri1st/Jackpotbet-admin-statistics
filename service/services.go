package service

import (
	"admin-stats/repository"
	"admin-stats/server/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

type Services struct {
	Transaction *TransactionService
	Stats       *StatsService
}

type ServicesOptions struct {
	Repos    *repository.Repos
	Log      logger.Logger
	Redis    *redis.Client
	CacheTTL time.Duration
}

func NewServices(opts *ServicesOptions) *Services {
	return &Services{
		Transaction: NewTransactionService(&TransactionServiceOptions{
			Repo: opts.Repos.Transaction,
			Log:  opts.Log,
		}),
		Stats: NewStatsService(&StatsServiceOptions{
			TxRepo:    opts.Repos.Transaction,
			StatsRepo: opts.Repos.DailyStats,
			Redis:     opts.Redis,
			CacheTTL:  opts.CacheTTL,
		}),
	}
}
