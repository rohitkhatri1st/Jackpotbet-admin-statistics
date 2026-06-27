package mongo

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
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

func (r *TransactionRepository) GetTransactions(ctx context.Context) ([]model.Transaction, error) {
	cursor, err := r.collection.Find(ctx, bson.D{})
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

// compile-time check: fails here if TransactionRepository is missing any interface method.
var _ repository.TransactionRepository = (*TransactionRepository)(nil)
