package service

import (
	"admin-stats/repository"
	"admin-stats/server/logger"
)

type Services struct {
	Transaction *TransactionService
}

type ServicesOptions struct {
	Repos *repository.Repos
	Log   logger.Logger
}

func NewServices(opts *ServicesOptions) *Services {
	return &Services{
		Transaction: NewTransactionService(&TransactionServiceOptions{
			Repo: opts.Repos.Transaction,
			Log:  opts.Log,
		}),
	}
}
