package mongo

import (
	"admin-stats/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type dailyWagerAggResult struct {
	ID struct {
		Date     string `bson:"date"`
		Currency string `bson:"currency"`
	} `bson:"_id"`
	TotalAmount bson.Decimal128 `bson:"totalAmount"`
	TotalUSD    bson.Decimal128 `bson:"totalUSD"`
}

func (r *TransactionRepository) GetDailyWagerVolume(ctx context.Context, filter repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
	pipeline := buildDailyWagerPipeline(filter)

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raw []dailyWagerAggResult
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, err
	}

	result := make([]repository.DailyWagerVolumeEntry, len(raw))
	for i, row := range raw {
		result[i] = repository.DailyWagerVolumeEntry{
			Date:      row.ID.Date,
			Currency:  row.ID.Currency,
			Volume:    row.TotalAmount.String(),
			VolumeUSD: row.TotalUSD.String(),
		}
	}
	return result, nil
}

func buildDailyWagerPipeline(filter repository.DailyWagerVolumeFilter) mongo.Pipeline {
	matchStage := bson.D{{Key: "type", Value: "Wager"}}
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
			{Key: "date", Value: bson.D{
				{Key: "$dateToString", Value: bson.D{
					{Key: "format", Value: "%Y-%m-%d"},
					{Key: "date", Value: "$createdAt"},
				}},
			}},
			{Key: "currency", Value: "$currency"},
		}},
		{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		{Key: "totalUSD", Value: bson.D{{Key: "$sum", Value: "$usdAmount"}}},
	}

	sortStage := bson.D{
		{Key: "_id.date", Value: 1},
		{Key: "_id.currency", Value: 1},
	}

	return mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: groupStage}},
		{{Key: "$sort", Value: sortStage}},
	}
}
