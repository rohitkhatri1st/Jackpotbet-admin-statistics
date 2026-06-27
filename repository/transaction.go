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

type WagerRankFilter struct {
	From *time.Time
	To   *time.Time
}

// UserWagerTotal holds a single user's total USD wagered for a period, sorted descending.
type UserWagerTotal struct {
	UserID   bson.ObjectID
	TotalUSD string // decimal string
}

type TransactionRepository interface {
	GetTransactions(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error)
	CreateTransaction(ctx context.Context, t model.Transaction) error
	BulkInsertTransactions(ctx context.Context, transactions []model.Transaction) error
	// ComputeDailyStats aggregates raw transactions into per-(date, currency) totals
	// for the given time window. Used by the cron and the manual recompute endpoint.
	ComputeDailyStats(ctx context.Context, from, to time.Time) ([]model.DailyStats, error)
	GetAllUserWagerTotals(ctx context.Context, filter WagerRankFilter) ([]UserWagerTotal, error)
}
