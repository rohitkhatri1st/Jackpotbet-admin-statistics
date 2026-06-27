package mongo

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const dailyStatsCollectionName = "daily_stats"

type DailyStatsRepository struct {
	collection *driver.Collection
}

func NewDailyStatsRepository(db *driver.Database) *DailyStatsRepository {
	return &DailyStatsRepository{
		collection: db.Collection(dailyStatsCollectionName),
	}
}

func (r *DailyStatsRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []driver.IndexModel{
		// Unique constraint: one document per (date, currency). Also covers date range queries.
		{
			Keys:    bson.D{{Key: "date", Value: 1}, {Key: "currency", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	return err
}

// Upsert writes pre-computed daily stats, overwriting any existing document for the same
// (date, currency) pair. Safe to call repeatedly — used by both the cron and recompute endpoint.
func (r *DailyStatsRepository) Upsert(ctx context.Context, stats []model.DailyStats) error {
	if len(stats) == 0 {
		return nil
	}
	models := make([]driver.WriteModel, len(stats))
	for i, s := range stats {
		filter := bson.D{
			{Key: "date", Value: s.Date},
			{Key: "currency", Value: s.Currency},
		}
		models[i] = driver.NewReplaceOneModel().
			SetFilter(filter).
			SetReplacement(s).
			SetUpsert(true)
	}
	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *DailyStatsRepository) GetByDateRange(ctx context.Context, filter repository.DailyStatsFilter) ([]model.DailyStats, error) {
	query := bson.D{}
	dateFilter := bson.D{}
	if filter.From != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: dateString(*filter.From)})
	}
	if filter.To != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: dateString(*filter.To)})
	}
	if len(dateFilter) > 0 {
		query = append(query, bson.E{Key: "date", Value: dateFilter})
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: 1}, {Key: "currency", Value: 1}})
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []model.DailyStats
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

// dateString formats a time.Time as "YYYY-MM-DD" to compare against the stored date field.
func dateString(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

var _ repository.DailyStatsRepository = (*DailyStatsRepository)(nil)
