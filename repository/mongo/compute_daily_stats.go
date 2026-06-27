package mongo

import (
	"admin-stats/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type dailyStatsAggResult struct {
	ID struct {
		Date     string `bson:"date"`
		Currency string `bson:"currency"`
		Type     string `bson:"type"`
	} `bson:"_id"`
	Total    bson.Decimal128 `bson:"total"`
	TotalUSD bson.Decimal128 `bson:"totalUSD"`
}

// ComputeDailyStats aggregates raw transactions for [from, to] into per-(date, currency)
// totals, combining Wager and Payout into a single DailyStats document each.
func (r *TransactionRepository) ComputeDailyStats(ctx context.Context, from, to time.Time) ([]model.DailyStats, error) {
	cursor, err := r.collection.Aggregate(ctx, buildComputeStatsPipeline(from, to))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raw []dailyStatsAggResult
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, err
	}

	return mergeIntoDailyStats(raw, time.Now().UTC()), nil
}

// mergeIntoDailyStats groups flat (date, currency, type) rows into one DailyStats
// per (date, currency), combining Wager and Payout totals.
func mergeIntoDailyStats(raw []dailyStatsAggResult, computedAt time.Time) []model.DailyStats {
	type key struct{ date, currency string }
	index := make(map[key]*model.DailyStats)
	order := make([]key, 0, len(raw))

	for _, row := range raw {
		k := key{date: row.ID.Date, currency: row.ID.Currency}
		entry, ok := index[k]
		if !ok {
			entry = &model.DailyStats{
				Date:       row.ID.Date,
				Currency:   row.ID.Currency,
				ComputedAt: computedAt,
			}
			index[k] = entry
			order = append(order, k)
		}
		switch row.ID.Type {
		case "Wager":
			entry.Wagers = row.Total
			entry.WagersUSD = row.TotalUSD
		case "Payout":
			entry.Payouts = row.Total
			entry.PayoutsUSD = row.TotalUSD
		}
	}

	result := make([]model.DailyStats, 0, len(order))
	for _, k := range order {
		result = append(result, *index[k])
	}
	return result
}

func buildComputeStatsPipeline(from, to time.Time) driver.Pipeline {
	matchStage := bson.D{
		{Key: "createdAt", Value: bson.D{
			{Key: "$gte", Value: from},
			{Key: "$lte", Value: to},
		}},
	}
	groupStage := bson.D{
		{Key: "_id", Value: bson.D{
			{Key: "date", Value: bson.D{
				{Key: "$dateToString", Value: bson.D{
					{Key: "format", Value: "%Y-%m-%d"},
					{Key: "date", Value: "$createdAt"},
				}},
			}},
			{Key: "currency", Value: "$currency"},
			{Key: "type", Value: "$type"},
		}},
		{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		{Key: "totalUSD", Value: bson.D{{Key: "$sum", Value: "$usdAmount"}}},
	}
	sortStage := bson.D{
		{Key: "_id.date", Value: 1},
		{Key: "_id.currency", Value: 1},
	}
	return driver.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: groupStage}},
		{{Key: "$sort", Value: sortStage}},
	}
}
