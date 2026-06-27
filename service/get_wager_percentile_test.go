package service_test

import (
	"admin-stats/repository"
	"admin-stats/service"
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetWagerPercentile(t *testing.T) {
	ctx := context.Background()
	userID := bson.NewObjectID()

	t.Run("nil input returns error", func(t *testing.T) {
		svc := newService(&mockRepo{})
		_, err := svc.GetWagerPercentile(ctx, nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})

	t.Run("rank 1 of 100 — top 1%", func(t *testing.T) {
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{Found: true, TotalUSD: "5000.00", Rank: 1, TotalUsers: 100}, nil
			},
		})

		got, err := svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Rank != 1 {
			t.Errorf("rank: got %d, want 1", got.Rank)
		}
		if got.TotalUsers != 100 {
			t.Errorf("totalUsers: got %d, want 100", got.TotalUsers)
		}
		// 1/100 * 100 = 1.00
		if got.TopPercentile != "1.00" {
			t.Errorf("topPercentile: got %q, want 1.00", got.TopPercentile)
		}
		assertDecimal(t, "totalUSD", got.TotalUSD, "5000.00")
	})

	t.Run("rank 10 of 500 — top 2%", func(t *testing.T) {
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{Found: true, TotalUSD: "1200.00", Rank: 10, TotalUsers: 500}, nil
			},
		})

		got, err := svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 10/500 * 100 = 2.00
		if got.TopPercentile != "2.00" {
			t.Errorf("topPercentile: got %q, want 2.00", got.TopPercentile)
		}
	})

	t.Run("user not found — placed last, topPercentile is 100%", func(t *testing.T) {
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				// 9 other users wagered; this user did not.
				return repository.UserWagerRank{Found: false, TotalUsers: 9}, nil
			},
		})

		got, err := svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// totalUsers becomes 10, user is placed at rank 10 → 10/10*100 = 100.00
		if got.Rank != 10 {
			t.Errorf("rank: got %d, want 10 (placed last)", got.Rank)
		}
		if got.TotalUsers != 10 {
			t.Errorf("totalUsers: got %d, want 10 (original 9 + the user themselves)", got.TotalUsers)
		}
		if got.TopPercentile != "100.00" {
			t.Errorf("topPercentile: got %q, want 100.00", got.TopPercentile)
		}
		assertDecimal(t, "totalUSD (no wagers)", got.TotalUSD, "0.00")
	})

	t.Run("user not found in empty period — placed last of 1", func(t *testing.T) {
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{Found: false, TotalUsers: 0}, nil
			},
		})

		got, err := svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Nobody wagered at all; user becomes rank 1 of 1.
		if got.Rank != 1 {
			t.Errorf("rank: got %d, want 1", got.Rank)
		}
		if got.TotalUsers != 1 {
			t.Errorf("totalUsers: got %d, want 1", got.TotalUsers)
		}
		if got.TopPercentile != "100.00" {
			t.Errorf("topPercentile: got %q, want 100.00", got.TopPercentile)
		}
	})

	t.Run("userID is passed through to repo", func(t *testing.T) {
		var capturedID bson.ObjectID
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, id bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				capturedID = id
				return repository.UserWagerRank{Found: true, TotalUSD: "100.00", Rank: 1, TotalUsers: 1}, nil
			},
		})

		_, _ = svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})

		if capturedID != userID {
			t.Errorf("userID passed to repo: got %s, want %s", capturedID, userID)
		}
	})

	t.Run("repo error is propagated", func(t *testing.T) {
		repoErr := errors.New("rank query failed")
		svc := newService(&mockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{}, repoErr
			},
		})

		_, err := svc.GetWagerPercentile(ctx, &service.GetWagerPercentileInput{UserID: userID})
		if !errors.Is(err, repoErr) {
			t.Errorf("error: got %v, want %v", err, repoErr)
		}
	})
}
