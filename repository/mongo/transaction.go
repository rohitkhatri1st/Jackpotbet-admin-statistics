package mongo

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "transactions"

// TransactionRepository implements repository.TransactionRepository using MongoDB.
type TransactionRepository struct {
	collection *driver.Collection
}

func NewTransactionRepository(db *driver.Database) *TransactionRepository {
	return &TransactionRepository{
		collection: db.Collection(collectionName),
	}
}

// EnsureIndexes creates the indexes required for efficient querying.
// Safe to call on every startup — MongoDB skips creation if the index already exists.
func (r *TransactionRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []driver.IndexModel{
		// Covers date-range $match in ComputeDailyStats and GetTransactions.
		{Keys: bson.D{{Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		// Compound index for wager percentile aggregation (filter by type + group by userId).
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "createdAt", Value: 1}}},
	})
	return err
}

func (r *TransactionRepository) GetAllUserWagerTotals(ctx context.Context, filter repository.WagerRankFilter) ([]repository.UserWagerTotal, error) {
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

	pipeline := driver.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "totalUSD", Value: bson.D{{Key: "$sum", Value: "$usdAmount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "totalUSD", Value: -1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type aggResult struct {
		UserID   bson.ObjectID   `bson:"_id"`
		TotalUSD bson.Decimal128 `bson:"totalUSD"`
	}
	var raw []aggResult
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, err
	}

	result := make([]repository.UserWagerTotal, len(raw))
	for i, row := range raw {
		result[i] = repository.UserWagerTotal{
			UserID:   row.UserID,
			TotalUSD: row.TotalUSD.String(),
		}
	}
	return result, nil
}

func (r *TransactionRepository) CreateTransaction(ctx context.Context, t model.Transaction) error {
	_, err := r.collection.InsertOne(ctx, t)
	return err
}

func (r *TransactionRepository) BulkInsertTransactions(ctx context.Context, transactions []model.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}
	docs := make([]any, len(transactions))
	for i, t := range transactions {
		docs[i] = t
	}
	_, err := r.collection.InsertMany(ctx, docs)
	return err
}

func (r *TransactionRepository) GetTransactions(ctx context.Context, filter repository.TransactionFilter) ([]model.Transaction, error) {
	query := buildGetTransactionsQuery(filter)
	opts := options.Find().SetLimit(filter.Limit).SetSort(bson.D{{Key: "_id", Value: 1}})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []model.Transaction
	if err = cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func buildGetTransactionsQuery(filter repository.TransactionFilter) bson.D {
	query := bson.D{}

	if filter.Cursor != nil {
		query = append(query, bson.E{Key: "_id", Value: bson.D{{Key: "$gt", Value: *filter.Cursor}}})
	}

	dateFilter := bson.D{}
	if filter.From != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: *filter.From})
	}
	if filter.To != nil {
		dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: *filter.To})
	}
	if len(dateFilter) > 0 {
		query = append(query, bson.E{Key: "createdAt", Value: dateFilter})
	}

	return query
}

// compile-time check: fails here if TransactionRepository is missing any interface method.
var _ repository.TransactionRepository = (*TransactionRepository)(nil)
