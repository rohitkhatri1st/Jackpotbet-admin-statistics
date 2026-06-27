package mongo

import (
	"admin-stats/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ggrAggResult is the shape of each document returned by the aggregation pipeline.
type ggrAggResult struct {
	ID struct {
		Currency string `bson:"currency"`
		Type     string `bson:"type"`
	} `bson:"_id"`
	TotalAmount bson.Decimal128 `bson:"totalAmount"`
	TotalUSD    bson.Decimal128 `bson:"totalUSD"`
}

func (r *TransactionRepository) GetGGR(ctx context.Context, filter repository.GGRFilter) ([]repository.CurrencyTotals, error) {
	pipeline := buildGGRPipeline(filter)

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raw []ggrAggResult
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, err
	}

	return groupByCurrency(raw), nil
}

// groupByCurrency converts the flat currency+type aggregation results into
// one CurrencyTotals entry per currency.
func groupByCurrency(raw []ggrAggResult) []repository.CurrencyTotals {
	index := make(map[string]*repository.CurrencyTotals, len(raw))
	for _, row := range raw {
		entry, ok := index[row.ID.Currency]
		if !ok {
			entry = &repository.CurrencyTotals{Currency: row.ID.Currency}
			index[row.ID.Currency] = entry
		}
		switch row.ID.Type {
		case "Wager":
			entry.Wagers = row.TotalAmount.String()
			entry.WagersUSD = row.TotalUSD.String()
		case "Payout":
			entry.Payouts = row.TotalAmount.String()
			entry.PayoutsUSD = row.TotalUSD.String()
		}
	}

	result := make([]repository.CurrencyTotals, 0, len(index))
	for _, v := range index {
		result = append(result, *v)
	}
	return result
}

func buildGGRPipeline(filter repository.GGRFilter) mongo.Pipeline {
	matchStage := bson.D{}
	dateFilter := bson.D{}
	if filter.From != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: *filter.From})
	}
	if filter.To != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: *filter.To})
	}
	if len(dateFilter) > 0 {
		matchStage = append(matchStage, bson.E{Key: "createdAt", Value: dateFilter})
	}

	groupStage := bson.D{
		{Key: "_id", Value: bson.D{
			{Key: "currency", Value: "$currency"},
			{Key: "type", Value: "$type"},
		}},
		{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		{Key: "totalUSD", Value: bson.D{{Key: "$sum", Value: "$usdAmount"}}},
	}

	return mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: groupStage}},
	}
}
