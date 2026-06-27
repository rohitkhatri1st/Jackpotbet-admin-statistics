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

// CurrencyTotals holds wager/payout sums for a single currency as decimal strings,
// ready for arithmetic in the service layer without any bson dependency.
type CurrencyTotals struct {
	Currency   string
	Wagers     string
	Payouts    string
	WagersUSD  string
	PayoutsUSD string
}

type GGRFilter struct {
	From *time.Time
	To   *time.Time
}

type TransactionRepository interface {
	GetTransactions(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error)
	CreateTransaction(ctx context.Context, t model.Transaction) error
	BulkInsertTransactions(ctx context.Context, transactions []model.Transaction) error
	GetGGR(ctx context.Context, filter GGRFilter) ([]CurrencyTotals, error)
}
