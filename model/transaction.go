package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Transaction struct {
	ID        bson.ObjectID   `bson:"_id"       json:"id"`
	CreatedAt time.Time       `bson:"createdAt" json:"createdAt"`
	UserID    bson.ObjectID   `bson:"userId"    json:"userId"`
	RoundID   string          `bson:"roundId"   json:"roundId"`
	Type      string          `bson:"type"      json:"type"`
	Amount    bson.Decimal128 `bson:"amount"    json:"amount"`
	Currency  string          `bson:"currency"  json:"currency"`
	USDAmount bson.Decimal128 `bson:"usdAmount" json:"usdAmount"`
}
