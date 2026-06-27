package service_test

import (
	"admin-stats/model"
	"admin-stats/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCreateTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("nil input returns error", func(t *testing.T) {
		svc := newService(&mockRepo{})
		_, err := svc.CreateTransaction(ctx, nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})

	t.Run("USD amount computed from static rate — USDT 1:1", func(t *testing.T) {
		var captured model.Transaction
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, tx model.Transaction) error {
				captured = tx
				return nil
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID:   bson.NewObjectID(),
			RoundID:  "round-1",
			Type:     "Wager",
			Currency: "USDT",
			Amount:   "250.00",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		usd, _ := decimal.NewFromString(captured.USDAmount.String())
		if !usd.Equal(decimal.NewFromInt(250)) {
			t.Errorf("usdAmount: got %s, want 250.00 (USDT rate is 1:1)", usd)
		}
	})

	t.Run("USD amount computed from static rate — ETH at $3000", func(t *testing.T) {
		var captured model.Transaction
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, tx model.Transaction) error {
				captured = tx
				return nil
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID:   bson.NewObjectID(),
			RoundID:  "round-1",
			Type:     "Wager",
			Currency: "ETH",
			Amount:   "2.00000000",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		usd, _ := decimal.NewFromString(captured.USDAmount.String())
		if !usd.Equal(decimal.NewFromInt(6000)) {
			t.Errorf("usdAmount: got %s, want 6000.00 (2 ETH × $3000)", usd)
		}
	})

	t.Run("createdAt defaults to current UTC when not supplied", func(t *testing.T) {
		before := time.Now().UTC()
		var captured model.Transaction
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, tx model.Transaction) error {
				captured = tx
				return nil
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID: bson.NewObjectID(), Type: "Wager", Currency: "USDT", Amount: "10.00",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		after := time.Now().UTC()

		if captured.CreatedAt.Before(before) || captured.CreatedAt.After(after) {
			t.Errorf("createdAt %v is outside [%v, %v]", captured.CreatedAt, before, after)
		}
		if captured.CreatedAt.Location() != time.UTC {
			t.Errorf("createdAt location: got %v, want UTC", captured.CreatedAt.Location())
		}
	})

	t.Run("caller-supplied createdAt is normalised to UTC", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/New_York")
		nyTime := time.Date(2024, 6, 15, 10, 30, 0, 0, loc)
		wantUTC := nyTime.UTC()

		var captured model.Transaction
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, tx model.Transaction) error {
				captured = tx
				return nil
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID: bson.NewObjectID(), Type: "Wager", Currency: "USDT", Amount: "10.00",
			CreatedAt: &nyTime,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !captured.CreatedAt.Equal(wantUTC) {
			t.Errorf("createdAt: got %v, want %v", captured.CreatedAt, wantUTC)
		}
		if captured.CreatedAt.Location() != time.UTC {
			t.Errorf("createdAt location: got %v, want UTC", captured.CreatedAt.Location())
		}
	})

	t.Run("all input fields passed through to repo", func(t *testing.T) {
		userID := bson.NewObjectID()
		var captured model.Transaction
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, tx model.Transaction) error {
				captured = tx
				return nil
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID: userID, RoundID: "round-xyz", Type: "Payout",
			Currency: "USDT", Amount: "99.00",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if captured.UserID != userID {
			t.Errorf("userID: got %s, want %s", captured.UserID, userID)
		}
		if captured.RoundID != "round-xyz" {
			t.Errorf("roundID: got %q, want round-xyz", captured.RoundID)
		}
		if captured.Type != "Payout" {
			t.Errorf("type: got %q, want Payout", captured.Type)
		}
		if captured.Currency != "USDT" {
			t.Errorf("currency: got %q, want USDT", captured.Currency)
		}
	})

	t.Run("unsupported currency returns error without calling repo", func(t *testing.T) {
		svc := newService(&mockRepo{
			// createTransactionFn left nil — will panic if called
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID: bson.NewObjectID(), Type: "Wager", Currency: "DOGE", Amount: "100.00",
		})
		if err == nil {
			t.Error("expected error for unsupported currency")
		}
	})

	t.Run("repo error is propagated", func(t *testing.T) {
		repoErr := errors.New("write failed")
		svc := newService(&mockRepo{
			createTransactionFn: func(_ context.Context, _ model.Transaction) error {
				return repoErr
			},
		})

		_, err := svc.CreateTransaction(ctx, &service.CreateTransactionInput{
			UserID: bson.NewObjectID(), Type: "Wager", Currency: "USDT", Amount: "10.00",
		})
		if !errors.Is(err, repoErr) {
			t.Errorf("error: got %v, want %v", err, repoErr)
		}
	})
}
