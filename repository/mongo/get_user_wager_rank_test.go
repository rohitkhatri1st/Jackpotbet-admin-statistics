//go:build integration

package mongo_test

import (
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetUserWagerRank(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	noFilter := repository.WagerRankFilter{}

	t.Run("rank 1 of 3", func(t *testing.T) {
		clearCollection(t)
		userA, userB, userC := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
		insertWager(t, userA, "1000.00", now)
		insertWager(t, userB, "500.00", now)
		insertWager(t, userC, "250.00", now)

		got, err := testRepo.GetUserWagerRank(ctx, userA, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true, got false")
		}
		if got.Rank != 1 {
			t.Errorf("rank: got %d, want 1", got.Rank)
		}
		if got.TotalUsers != 3 {
			t.Errorf("totalUsers: got %d, want 3", got.TotalUsers)
		}
		assertDecimalEqual(t, "totalUSD", got.TotalUSD, "1000.00")
	})

	t.Run("rank 3 of 3", func(t *testing.T) {
		clearCollection(t)
		userA, userB, userC := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
		insertWager(t, userA, "1000.00", now)
		insertWager(t, userB, "500.00", now)
		insertWager(t, userC, "250.00", now)

		got, err := testRepo.GetUserWagerRank(ctx, userC, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true, got false")
		}
		if got.Rank != 3 {
			t.Errorf("rank: got %d, want 3", got.Rank)
		}
		if got.TotalUsers != 3 {
			t.Errorf("totalUsers: got %d, want 3", got.TotalUsers)
		}
	})

	t.Run("multiple wagers per user — rank uses summed total", func(t *testing.T) {
		clearCollection(t)
		userA, userB := bson.NewObjectID(), bson.NewObjectID()
		// userA: three transactions totalling 900
		insertWager(t, userA, "300.00", now)
		insertWager(t, userA, "300.00", now)
		insertWager(t, userA, "300.00", now)
		// userB: one transaction of 1000 — ranks higher than userA
		insertWager(t, userB, "1000.00", now)

		got, err := testRepo.GetUserWagerRank(ctx, userA, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true")
		}
		if got.Rank != 2 {
			t.Errorf("rank: got %d, want 2", got.Rank)
		}
		assertDecimalEqual(t, "totalUSD", got.TotalUSD, "900.00")
	})

	t.Run("user never wagered — Found=false", func(t *testing.T) {
		clearCollection(t)
		userA, outsider := bson.NewObjectID(), bson.NewObjectID()
		insertWager(t, userA, "500.00", now)

		got, err := testRepo.GetUserWagerRank(ctx, outsider, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Found {
			t.Error("expected Found=false for a user who never wagered")
		}
		if got.TotalUsers != 1 {
			t.Errorf("totalUsers: got %d, want 1", got.TotalUsers)
		}
	})

	t.Run("all of user's wagers are outside the date range — Found=false", func(t *testing.T) {
		clearCollection(t)
		userA, userB := bson.NewObjectID(), bson.NewObjectID()
		twoYearsAgo := now.AddDate(-2, 0, 0)
		insertWager(t, userA, "1000.00", twoYearsAgo) // outside filter
		insertWager(t, userB, "500.00", now)           // inside filter

		from := now.AddDate(-1, 0, 0)
		filter := repository.WagerRankFilter{From: &from}

		got, err := testRepo.GetUserWagerRank(ctx, userA, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Found {
			t.Error("expected Found=false when user's wagers are all outside the date range")
		}
		if got.TotalUsers != 1 {
			t.Errorf("totalUsers: got %d, want 1 (only userB is in range)", got.TotalUsers)
		}
	})

	t.Run("out-of-range wagers excluded — rank uses only in-range sum", func(t *testing.T) {
		clearCollection(t)
		userA, userB := bson.NewObjectID(), bson.NewObjectID()
		from := now.Add(-48 * time.Hour)
		to := now.Add(-24 * time.Hour)
		inWindow := now.Add(-36 * time.Hour)

		insertWager(t, userA, "100.00", inWindow) // in range
		insertWager(t, userA, "99999.00", now)    // after 'to' — must be ignored
		insertWager(t, userB, "500.00", inWindow) // userB ranks 1 in range

		filter := repository.WagerRankFilter{From: &from, To: &to}
		got, err := testRepo.GetUserWagerRank(ctx, userA, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true for in-range wagers")
		}
		if got.Rank != 2 {
			t.Errorf("rank: got %d, want 2 (userB has higher in-range total)", got.Rank)
		}
		assertDecimalEqual(t, "totalUSD", got.TotalUSD, "100.00")
	})

	t.Run("payout transactions are ignored", func(t *testing.T) {
		clearCollection(t)
		userA, userB := bson.NewObjectID(), bson.NewObjectID()
		insertWager(t, userA, "200.00", now)
		insertPayout(t, userB, "9999.00", now) // large payout but no wager — must not appear

		got, err := testRepo.GetUserWagerRank(ctx, userA, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true")
		}
		if got.Rank != 1 {
			t.Errorf("rank: got %d, want 1 (payout-only user must be excluded)", got.Rank)
		}
		if got.TotalUsers != 1 {
			t.Errorf("totalUsers: got %d, want 1 (payout-only user must not count)", got.TotalUsers)
		}
	})

	t.Run("tied users share rank — next user skips a position", func(t *testing.T) {
		clearCollection(t)
		userA, userB, userC := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
		insertWager(t, userA, "500.00", now)
		insertWager(t, userB, "500.00", now) // tied with userA — both rank 1
		insertWager(t, userC, "100.00", now) // MongoDB $rank skips to 3 after a two-way tie

		gotA, err := testRepo.GetUserWagerRank(ctx, userA, noFilter)
		if err != nil {
			t.Fatalf("userA: %v", err)
		}
		gotC, err := testRepo.GetUserWagerRank(ctx, userC, noFilter)
		if err != nil {
			t.Fatalf("userC: %v", err)
		}

		if gotA.Rank != 1 {
			t.Errorf("userA rank: got %d, want 1", gotA.Rank)
		}
		// $rank skips rank 2 because two users share rank 1.
		if gotC.Rank != 3 {
			t.Errorf("userC rank: got %d, want 3 (rank 2 skipped due to tie)", gotC.Rank)
		}
		if gotA.TotalUsers != 3 {
			t.Errorf("totalUsers: got %d, want 3", gotA.TotalUsers)
		}
	})

	t.Run("single user — rank 1 of 1", func(t *testing.T) {
		clearCollection(t)
		userA := bson.NewObjectID()
		insertWager(t, userA, "750.00", now)

		got, err := testRepo.GetUserWagerRank(ctx, userA, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Found {
			t.Fatal("expected Found=true")
		}
		if got.Rank != 1 {
			t.Errorf("rank: got %d, want 1", got.Rank)
		}
		if got.TotalUsers != 1 {
			t.Errorf("totalUsers: got %d, want 1", got.TotalUsers)
		}
	})

	t.Run("empty collection — not found, zero total users", func(t *testing.T) {
		clearCollection(t)

		got, err := testRepo.GetUserWagerRank(ctx, bson.NewObjectID(), noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Found {
			t.Error("expected Found=false on an empty collection")
		}
		if got.TotalUsers != 0 {
			t.Errorf("totalUsers: got %d, want 0", got.TotalUsers)
		}
	})
}
