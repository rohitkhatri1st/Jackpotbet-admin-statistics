package repository

import (
	"admin-stats/model"
	"context"
)

type TransactionRepository interface {
	GetTransactions(ctx context.Context) ([]model.Transaction, error)
}
