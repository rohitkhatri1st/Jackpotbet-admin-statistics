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

type DailyWagerVolumeFilter struct {
	From *time.Time
	To   *time.Time
}

// DailyWagerVolumeEntry holds the wager volume for one (date, currency) bucket.
type DailyWagerVolumeEntry struct {
	Date      string // "2006-01-02"
	Currency  string
	Volume    string // decimal string
	VolumeUSD string // decimal string
}

type WagerRankFilter struct {
	From *time.Time
	To   *time.Time
}

// UserWagerRank holds the result of a single user's wager rank lookup.
// Found is false when the user had no wagers in the requested period.
type UserWagerRank struct {
	Found      bool
	TotalUSD   string // decimal string; meaningful only when Found is true
	Rank       int    // 1-indexed position; meaningful only when Found is true
	TotalUsers int    // total distinct users who wagered in the period
}

type TransactionRepository interface {
	GetTransactions(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error)
	CreateTransaction(ctx context.Context, t model.Transaction) error
	BulkInsertTransactions(ctx context.Context, transactions []model.Transaction) error
	GetGGR(ctx context.Context, filter GGRFilter) ([]CurrencyTotals, error)
	GetDailyWagerVolume(ctx context.Context, filter DailyWagerVolumeFilter) ([]DailyWagerVolumeEntry, error)
	// GetUserWagerRank returns the requesting user's rank among all users by total USD
	// wagered in the given period. MongoDB assigns the rank via $setWindowFields so only
	// one document is returned to the application regardless of how many users exist.
	GetUserWagerRank(ctx context.Context, userID bson.ObjectID, filter WagerRankFilter) (UserWagerRank, error)
}
