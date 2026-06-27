package repository

import (
	"admin-stats/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type TransactionFilter struct {
	From   *time.Time
	To     *time.Time
	Cursor *bson.ObjectID
	Limit  int64
}

type TransactionRepository interface {
	GetTransactions(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error)
	CreateTransaction(ctx context.Context, t model.Transaction) error
	BulkInsertTransactions(ctx context.Context, transactions []model.Transaction) error
}
