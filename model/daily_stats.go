package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DailyStats stores pre-computed wager and payout totals per (date, currency).
// One document per calendar day per currency — populated nightly by the cron job
// and used by the GGR and daily wager volume endpoints.
type DailyStats struct {
	ID         bson.ObjectID   `bson:"_id,omitempty"`
	Date       string          `bson:"date"`       // "2006-01-02"
	Currency   string          `bson:"currency"`
	Wagers     bson.Decimal128 `bson:"wagers"`
	WagersUSD  bson.Decimal128 `bson:"wagersUSD"`
	Payouts    bson.Decimal128 `bson:"payouts"`
	PayoutsUSD bson.Decimal128 `bson:"payoutsUSD"`
	ComputedAt time.Time       `bson:"computedAt"`
}
