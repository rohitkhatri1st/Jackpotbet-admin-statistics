package repository

import (
	"admin-stats/model"
	"context"
)

type TransactionRepository interface {
	GetTransactions(ctx context.Context) ([]model.Transaction, error)
	CreateTransaction(ctx context.Context, t model.Transaction) error
}
