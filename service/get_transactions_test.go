package service_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/service"
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil input returns error", func(t *testing.T) {
		svc := newService(&mockRepo{})
		_, err := svc.GetTransactions(ctx, nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})

	t.Run("repo receives limit+1", func(t *testing.T) {
		var capturedLimit int64
		svc := newService(&mockRepo{
			getTransactionsFn: func(_ context.Context, f repository.TransactionFilter) ([]model.Transaction, error) {
				capturedLimit = f.Limit
				return nil, nil
			},
		})

		_, _ = svc.GetTransactions(ctx, &service.GetTransactionsInput{Limit: 5})

		if capturedLimit != 6 {
			t.Errorf("repo limit: got %d, want 6 (input+1)", capturedLimit)
		}
	})

	t.Run("next cursor set when repo returns limit+1 docs", func(t *testing.T) {
		ids := make([]bson.ObjectID, 6)
		for i := range ids {
			ids[i] = bson.NewObjectID()
		}
		txs := make([]model.Transaction, 6)
		for i, id := range ids {
			txs[i] = model.Transaction{ID: id}
		}

		svc := newService(&mockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return txs, nil
			},
		})

		got, err := svc.GetTransactions(ctx, &service.GetTransactionsInput{Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 5 {
			t.Errorf("data len: got %d, want 5 (trimmed to limit)", len(got.Data))
		}
		if got.Cursor == nil {
			t.Fatal("cursor: got nil, want non-nil (more pages exist)")
		}
		// Cursor must point to the last returned document, not the extra one.
		if *got.Cursor != ids[4] {
			t.Errorf("cursor: got %s, want %s (last of trimmed page)", got.Cursor, ids[4])
		}
	})

	t.Run("no cursor when repo returns exactly limit docs", func(t *testing.T) {
		txs := make([]model.Transaction, 5)
		for i := range txs {
			txs[i] = model.Transaction{ID: bson.NewObjectID()}
		}

		svc := newService(&mockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return txs, nil
			},
		})

		got, err := svc.GetTransactions(ctx, &service.GetTransactionsInput{Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 5 {
			t.Errorf("data len: got %d, want 5", len(got.Data))
		}
		if got.Cursor != nil {
			t.Errorf("cursor: got %s, want nil (no more pages)", got.Cursor)
		}
	})

	t.Run("no cursor on empty result", func(t *testing.T) {
		svc := newService(&mockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return nil, nil
			},
		})

		got, err := svc.GetTransactions(ctx, &service.GetTransactionsInput{Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Cursor != nil {
			t.Error("cursor: got non-nil, want nil on empty result")
		}
	})

	t.Run("repo error is propagated", func(t *testing.T) {
		repoErr := errors.New("db unavailable")
		svc := newService(&mockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.GetTransactions(ctx, &service.GetTransactionsInput{Limit: 10})
		if !errors.Is(err, repoErr) {
			t.Errorf("error: got %v, want %v", err, repoErr)
		}
	})
}
