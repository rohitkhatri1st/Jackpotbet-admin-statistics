package mongo

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "transactions"

// TransactionRepository implements repository.TransactionRepository using MongoDB.
type TransactionRepository struct {
	collection *mongo.Collection
}

func NewTransactionRepository(db *mongo.Database) *TransactionRepository {
	return &TransactionRepository{
		collection: db.Collection(collectionName),
	}
}

// EnsureIndexes creates the indexes required for efficient querying.
// Safe to call on every startup — MongoDB skips creation if the index already exists.
func (r *TransactionRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// Covers date-range filtering in GetTransactions and GetGGR $match stage.
		{Keys: bson.D{{Key: "createdAt", Value: 1}}},

		{Keys: bson.D{{Key: "userId", Value: 1}}},

		// Covers the GetAllUserWagerTotals $match stage (type + date range).
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "createdAt", Value: 1}}},
	})
	return err
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

// wagerRankFacetResult is the single document returned by the $facet stage.
type wagerRankFacetResult struct {
	UserRank []struct {
		TotalUSD bson.Decimal128 `bson:"totalUSD"`
		Rank     int32           `bson:"rank"`
	} `bson:"userRank"`
	Total []struct {
		N int `bson:"n"`
	} `bson:"total"`
}

func (r *TransactionRepository) GetUserWagerRank(ctx context.Context, userID bson.ObjectID, filter repository.WagerRankFilter) (repository.UserWagerRank, error) {
	pipeline := buildUserWagerRankPipeline(userID, filter)

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return repository.UserWagerRank{}, err
	}
	defer cursor.Close(ctx)

	// $facet always returns exactly one document.
	var results []wagerRankFacetResult
	if err := cursor.All(ctx, &results); err != nil {
		return repository.UserWagerRank{}, err
	}
	if len(results) == 0 {
		return repository.UserWagerRank{}, nil
	}

	facet := results[0]
	totalUsers := 0
	if len(facet.Total) > 0 {
		totalUsers = facet.Total[0].N
	}

	if len(facet.UserRank) == 0 {
		return repository.UserWagerRank{TotalUsers: totalUsers}, nil
	}

	u := facet.UserRank[0]
	return repository.UserWagerRank{
		Found:      true,
		TotalUSD:   u.TotalUSD.String(),
		Rank:       int(u.Rank),
		TotalUsers: totalUsers,
	}, nil
}

// buildUserWagerRankPipeline constructs the aggregation that computes a single user's
// wager rank without returning all users to the application:
//
//  1. $match — filter to Wager transactions within the date range.
//  2. $group — sum usdAmount per userId.
//  3. $facet — runs two sub-pipelines on the grouped result in parallel:
//     • "userRank": assigns rank via $setWindowFields (sorted by totalUSD desc),
//     then filters to the target user — returns at most one document.
//     • "total": counts distinct users who wagered.
func buildUserWagerRankPipeline(userID bson.ObjectID, filter repository.WagerRankFilter) mongo.Pipeline {
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

	// We can use $rowNumber instead of $rank if we want to break ties arbitrarily instead of assigning the same rank to tied users.
	rankSubPipeline := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "totalUSD", Value: -1}}},
			{Key: "output", Value: bson.D{
				{Key: "rank", Value: bson.D{{Key: "$rank", Value: bson.D{}}}},
			}},
		}}},
		bson.D{{Key: "$match", Value: bson.D{{Key: "_id", Value: userID}}}},
	}

	return mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "totalUSD", Value: bson.D{{Key: "$sum", Value: "$usdAmount"}}},
		}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "userRank", Value: rankSubPipeline},
			{Key: "total", Value: bson.A{bson.D{{Key: "$count", Value: "n"}}}},
		}}},
	}
}

// compile-time check: fails here if TransactionRepository is missing any interface method.
var _ repository.TransactionRepository = (*TransactionRepository)(nil)
