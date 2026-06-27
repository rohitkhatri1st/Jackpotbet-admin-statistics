//go:build integration

package mongo_test

import (
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetTransactions(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	user := bson.NewObjectID()

	t.Run("empty collection — returns empty slice", func(t *testing.T) {
		clearCollection(t)

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len: got %d, want 0", len(got))
		}
	})

	t.Run("results are sorted by _id ascending", func(t *testing.T) {
		clearCollection(t)
		// ObjectIDs are monotonically increasing — inserting A then B means A < B.
		insertWager(t, user, "100.00", now)
		insertWager(t, user, "200.00", now)
		insertWager(t, user, "300.00", now)

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len: got %d, want 3", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i].ID.String() <= got[i-1].ID.String() {
				t.Errorf("row[%d].ID (%s) is not > row[%d].ID (%s) — not sorted asc",
					i, got[i].ID, i-1, got[i-1].ID)
			}
		}
	})

	t.Run("limit is respected", func(t *testing.T) {
		clearCollection(t)
		insertWager(t, user, "100.00", now)
		insertWager(t, user, "200.00", now)
		insertWager(t, user, "300.00", now)

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 2})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("len: got %d, want 2 (limit applied)", len(got))
		}
	})

	t.Run("cursor skips to next page — $gt is exclusive", func(t *testing.T) {
		clearCollection(t)
		insertWager(t, user, "100.00", now)
		insertWager(t, user, "200.00", now)
		insertWager(t, user, "300.00", now)

		// Page 1: first two documents.
		page1, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 2})
		if err != nil {
			t.Fatalf("page1: %v", err)
		}
		if len(page1) != 2 {
			t.Fatalf("page1 len: got %d, want 2", len(page1))
		}

		// Use the last ID from page 1 as the cursor.
		cursor := page1[len(page1)-1].ID
		page2, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10, Cursor: &cursor})
		if err != nil {
			t.Fatalf("page2: %v", err)
		}

		// $gt is exclusive — the cursor document itself must not appear on page 2.
		if len(page2) != 1 {
			t.Fatalf("page2 len: got %d, want 1 (one doc remains after cursor)", len(page2))
		}
		if page2[0].ID == cursor {
			t.Error("cursor document appeared on page 2 — $gt must be exclusive")
		}
	})

	t.Run("full pagination walk — every document appears exactly once", func(t *testing.T) {
		clearCollection(t)
		for range 5 {
			insertWager(t, user, "100.00", now)
		}

		seen := make(map[bson.ObjectID]bool)
		var cursor *bson.ObjectID
		pageSize := int64(2)

		for {
			page, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: pageSize, Cursor: cursor})
			if err != nil {
				t.Fatalf("pagination walk: %v", err)
			}
			if len(page) == 0 {
				break
			}
			for _, tx := range page {
				if seen[tx.ID] {
					t.Errorf("duplicate document %s across pages", tx.ID)
				}
				seen[tx.ID] = true
			}
			last := page[len(page)-1].ID
			cursor = &last
		}

		if len(seen) != 5 {
			t.Errorf("total docs seen: got %d, want 5", len(seen))
		}
	})

	t.Run("cursor past last document — returns empty slice", func(t *testing.T) {
		clearCollection(t)
		insertWager(t, user, "100.00", now)

		all, _ := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10})
		last := all[len(all)-1].ID

		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10, Cursor: &last})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len: got %d, want 0 (no documents after last cursor)", len(got))
		}
	})

	t.Run("date range: from filter — older transactions excluded", func(t *testing.T) {
		clearCollection(t)
		twoYearsAgo := now.AddDate(-2, 0, 0)
		insertWager(t, user, "100.00", twoYearsAgo) // before from — excluded
		insertWager(t, user, "200.00", now)

		from := now.AddDate(-1, 0, 0)
		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10, From: &from})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1", len(got))
		}
		assertDecimalEqual(t, "usdAmount (in-range only)", got[0].USDAmount.String(), "200.00")
	})

	t.Run("date range: to filter — newer transactions excluded", func(t *testing.T) {
		clearCollection(t)
		past := now.AddDate(-1, 0, 0)
		insertWager(t, user, "100.00", past)
		insertWager(t, user, "200.00", now) // after to — excluded

		to := now.Add(-1 * time.Hour)
		got, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10, To: &to})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1", len(got))
		}
		assertDecimalEqual(t, "usdAmount (before to only)", got[0].USDAmount.String(), "100.00")
	})

	t.Run("cursor and date filter combine — cursor scopes within filtered results", func(t *testing.T) {
		clearCollection(t)
		recent := now.Add(-30 * time.Minute)
		old := now.AddDate(-2, 0, 0)

		insertWager(t, user, "100.00", recent) // in range
		insertWager(t, user, "200.00", recent) // in range
		insertWager(t, user, "300.00", old)    // out of range — must never appear

		from := now.AddDate(-1, 0, 0)
		page1, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 1, From: &from})
		if err != nil {
			t.Fatalf("page1: %v", err)
		}
		if len(page1) != 1 {
			t.Fatalf("page1 len: got %d, want 1", len(page1))
		}

		cursor := page1[0].ID
		page2, err := testRepo.GetTransactions(ctx, repository.TransactionFilter{Limit: 10, From: &from, Cursor: &cursor})
		if err != nil {
			t.Fatalf("page2: %v", err)
		}
		// Only one more in-range document should remain; the old one must not appear.
		if len(page2) != 1 {
			t.Fatalf("page2 len: got %d, want 1", len(page2))
		}
		if page2[0].ID == cursor {
			t.Error("cursor document must not appear on page 2")
		}
	})
}
