package service

import (
	"admin-stats/repository"
	"admin-stats/server/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

type Services struct {
	Transaction *TransactionService
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
			Repo:     opts.Repos.Transaction,
			Log:      opts.Log,
			Redis:    opts.Redis,
			CacheTTL: opts.CacheTTL,
		}),
	}
}
