//go:build integration

package mongo_test

import (
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetGGR(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	noFilter := repository.GGRFilter{}
	user := bson.NewObjectID()

	t.Run("empty collection — returns empty slice", func(t *testing.T) {
		clearCollection(t)

		got, err := testRepo.GetGGR(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len: got %d, want 0", len(got))
		}
	})

	t.Run("single currency — wagers and payouts summed correctly", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "100.00", "100.00", now)
		insertTx(t, "Wager", user, "USDT", "50.00", "50.00", now)
		insertTx(t, "Payout", user, "USDT", "80.00", "80.00", now)

		got, err := testRepo.GetGGR(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1", len(got))
		}
		entry, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found in result")
		}
		assertDecimalEqual(t, "USDT wagers", entry.Wagers, "150.00")
		assertDecimalEqual(t, "USDT wagersUSD", entry.WagersUSD, "150.00")
		assertDecimalEqual(t, "USDT payouts", entry.Payouts, "80.00")
		assertDecimalEqual(t, "USDT payoutsUSD", entry.PayoutsUSD, "80.00")
	})

	t.Run("currency with wagers only — payouts fields are empty", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "200.00", "200.00", now)

		got, err := testRepo.GetGGR(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entry, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		assertDecimalEqual(t, "USDT wagers", entry.Wagers, "200.00")
		// No payout document was grouped — fields stay at their zero string value.
		if entry.Payouts != "" {
			t.Errorf("payouts: got %q, want empty string (no payouts inserted)", entry.Payouts)
		}
		if entry.PayoutsUSD != "" {
			t.Errorf("payoutsUSD: got %q, want empty string", entry.PayoutsUSD)
		}
	})

	t.Run("currency with payouts only — wagers fields are empty", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Payout", user, "USDT", "300.00", "300.00", now)

		got, err := testRepo.GetGGR(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entry, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		if entry.Wagers != "" {
			t.Errorf("wagers: got %q, want empty string (no wagers inserted)", entry.Wagers)
		}
		assertDecimalEqual(t, "USDT payouts", entry.Payouts, "300.00")
	})

	t.Run("multiple currencies — each has independent totals", func(t *testing.T) {
		clearCollection(t)
		// ETH: 2 ETH wagered at $3000/ETH = $6000; 1 ETH payout = $3000
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", now)
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", now)
		insertTx(t, "Payout", user, "ETH", "1.00000000", "3000.00", now)
		// USDT: 500 wagered, 200 paid out
		insertTx(t, "Wager", user, "USDT", "500.00", "500.00", now)
		insertTx(t, "Payout", user, "USDT", "200.00", "200.00", now)

		got, err := testRepo.GetGGR(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len: got %d, want 2 (ETH + USDT)", len(got))
		}

		eth, ok := findCurrency(got, "ETH")
		if !ok {
			t.Fatal("ETH entry not found")
		}
		assertDecimalEqual(t, "ETH wagers (native)", eth.Wagers, "2.00000000")
		assertDecimalEqual(t, "ETH wagersUSD", eth.WagersUSD, "6000.00")
		assertDecimalEqual(t, "ETH payouts (native)", eth.Payouts, "1.00000000")
		assertDecimalEqual(t, "ETH payoutsUSD", eth.PayoutsUSD, "3000.00")

		usdt, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		assertDecimalEqual(t, "USDT wagers", usdt.Wagers, "500.00")
		assertDecimalEqual(t, "USDT payouts", usdt.Payouts, "200.00")
	})

	t.Run("date range: from filter — older transactions excluded", func(t *testing.T) {
		clearCollection(t)
		twoYearsAgo := now.AddDate(-2, 0, 0)
		insertTx(t, "Wager", user, "USDT", "9999.00", "9999.00", twoYearsAgo) // must be excluded
		insertTx(t, "Wager", user, "USDT", "100.00", "100.00", now)

		from := now.AddDate(-1, 0, 0)
		got, err := testRepo.GetGGR(ctx, repository.GGRFilter{From: &from})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entry, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		assertDecimalEqual(t, "wagers (only in-range)", entry.Wagers, "100.00")
	})

	t.Run("date range: from+to filter — only in-window transactions counted", func(t *testing.T) {
		clearCollection(t)
		from := now.Add(-48 * time.Hour)
		to := now.Add(-24 * time.Hour)
		inWindow := now.Add(-36 * time.Hour)

		insertTx(t, "Wager", user, "USDT", "500.00", "500.00", inWindow) // in range
		insertTx(t, "Wager", user, "USDT", "999.00", "999.00", now)      // after 'to' — excluded
		insertTx(t, "Payout", user, "USDT", "200.00", "200.00", inWindow)

		got, err := testRepo.GetGGR(ctx, repository.GGRFilter{From: &from, To: &to})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entry, ok := findCurrency(got, "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		assertDecimalEqual(t, "wagers (in-window only)", entry.Wagers, "500.00")
		assertDecimalEqual(t, "payouts (in-window only)", entry.Payouts, "200.00")
	})

	t.Run("date range: currency fully outside range — not in result", func(t *testing.T) {
		clearCollection(t)
		twoYearsAgo := now.AddDate(-2, 0, 0)
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", twoYearsAgo) // outside range
		insertTx(t, "Wager", user, "USDT", "100.00", "100.00", now)

		from := now.AddDate(-1, 0, 0)
		got, err := testRepo.GetGGR(ctx, repository.GGRFilter{From: &from})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := findCurrency(got, "ETH"); ok {
			t.Error("ETH must not appear in result — all its transactions are outside the date range")
		}
		if _, ok := findCurrency(got, "USDT"); !ok {
			t.Error("USDT must appear in result")
		}
	})
}
