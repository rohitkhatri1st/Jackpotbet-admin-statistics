//go:build integration

package mongo_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestEnsureIndexes(t *testing.T) {
	ctx := context.Background()

	t.Run("idempotent — calling twice returns no error", func(t *testing.T) {
		if err := testRepo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("first call: %v", err)
		}
		if err := testRepo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("second call: %v", err)
		}
	})
}

func TestCreateTransaction(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("inserted document is retrievable with correct fields", func(t *testing.T) {
		clearCollection(t)
		userID := bson.NewObjectID()
		tx := model.Transaction{
			ID:        bson.NewObjectID(),
			UserID:    userID,
			RoundID:   "round-abc",
			Type:      "Wager",
			Amount:    decimal128("1.50000000"),
			Currency:  "ETH",
			USDAmount: decimal128("4500.00"),
			CreatedAt: now.Truncate(time.Millisecond), // MongoDB stores ms precision
		}

		if err := testRepo.CreateTransaction(ctx, tx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1", len(got))
		}

		r := got[0]
		if r.ID != tx.ID {
			t.Errorf("ID: got %s, want %s", r.ID, tx.ID)
		}
		if r.UserID != userID {
			t.Errorf("UserID: got %s, want %s", r.UserID, userID)
		}
		if r.RoundID != "round-abc" {
			t.Errorf("RoundID: got %q, want %q", r.RoundID, "round-abc")
		}
		if r.Type != "Wager" {
			t.Errorf("Type: got %q, want Wager", r.Type)
		}
		if r.Currency != "ETH" {
			t.Errorf("Currency: got %q, want ETH", r.Currency)
		}
		assertDecimalEqual(t, "amount", r.Amount.String(), "1.50000000")
		assertDecimalEqual(t, "usdAmount", r.USDAmount.String(), "4500.00")
		if !r.CreatedAt.Equal(tx.CreatedAt) {
			t.Errorf("CreatedAt: got %v, want %v", r.CreatedAt, tx.CreatedAt)
		}
	})
}

func TestBulkInsertTransactions(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	user := bson.NewObjectID()

	t.Run("empty batch — no error, no documents inserted", func(t *testing.T) {
		clearCollection(t)

		if err := testRepo.BulkInsertTransactions(ctx, nil); err != nil {
			t.Fatalf("nil batch: %v", err)
		}
		if err := testRepo.BulkInsertTransactions(ctx, []model.Transaction{}); err != nil {
			t.Fatalf("empty slice: %v", err)
		}

		got, _ := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})
		if len(got) != 0 {
			t.Errorf("len: got %d, want 0 after empty inserts", len(got))
		}
	})

	t.Run("all documents in the batch are inserted", func(t *testing.T) {
		clearCollection(t)
		batch := make([]model.Transaction, 5)
		for i := range batch {
			batch[i] = model.Transaction{
				ID:        bson.NewObjectID(),
				UserID:    user,
				RoundID:   "round-bulk",
				Type:      "Wager",
				Amount:    decimal128("10.00"),
				Currency:  "USDT",
				USDAmount: decimal128("10.00"),
				CreatedAt: now,
			}
		}

		if err := testRepo.BulkInsertTransactions(ctx, batch); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if len(got) != 5 {
			t.Errorf("len: got %d, want 5", len(got))
		}
	})

	t.Run("IDs assigned in batch are preserved", func(t *testing.T) {
		clearCollection(t)
		ids := []bson.ObjectID{bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()}
		batch := make([]model.Transaction, len(ids))
		for i, id := range ids {
			batch[i] = model.Transaction{
				ID:        id,
				UserID:    user,
				RoundID:   "round-id-check",
				Type:      "Wager",
				Amount:    decimal128("5.00"),
				Currency:  "USDT",
				USDAmount: decimal128("5.00"),
				CreatedAt: now,
			}
		}

		if err := testRepo.BulkInsertTransactions(ctx, batch); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})
		gotIDs := make(map[bson.ObjectID]bool, len(got))
		for _, tx := range got {
			gotIDs[tx.ID] = true
		}
		for _, id := range ids {
			if !gotIDs[id] {
				t.Errorf("ID %s from batch not found in collection", id)
			}
		}
	})
}
