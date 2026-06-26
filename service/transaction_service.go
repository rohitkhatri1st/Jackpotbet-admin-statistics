package service

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"context"
)

type TransactionService struct {
	repo repository.TransactionRepository
	log  logger.Logger
}

type TransactionServiceOptions struct {
	Repo repository.TransactionRepository
	Log  logger.Logger
}

func NewTransactionService(opts *TransactionServiceOptions) *TransactionService {
	return &TransactionService{
		repo: opts.Repo,
		log:  opts.Log,
	}
}

func (s *TransactionService) GetTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.repo.GetTransactions(ctx)
}
