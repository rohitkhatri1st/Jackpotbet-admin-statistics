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

func (r *TransactionRepository) CreateTransaction(ctx context.Context, t model.Transaction) error {
	_, err := r.collection.InsertOne(ctx, t)
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
