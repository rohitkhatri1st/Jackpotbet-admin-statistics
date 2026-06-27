//go:build integration

package mongo_test

import (
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Fixed UTC timestamps on distinct calendar days. Using absolute times avoids
// flakiness when tests run near a UTC midnight boundary.
var (
	day1 = time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	day2 = time.Date(2024, 1, 11, 12, 0, 0, 0, time.UTC)
	day3 = time.Date(2024, 1, 12, 12, 0, 0, 0, time.UTC)
)

func TestGetDailyWagerVolume(t *testing.T) {
	ctx := context.Background()
	noFilter := repository.DailyWagerVolumeFilter{}
	user := bson.NewObjectID()

	t.Run("empty collection — returns empty slice", func(t *testing.T) {
		clearCollection(t)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len: got %d, want 0", len(got))
		}
	})

	t.Run("single day, single currency — volume and volumeUSD correct", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "150.00", "150.00", day1)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1", len(got))
		}
		entry, ok := findDailyEntry(got, dateStr(day1), "USDT")
		if !ok {
			t.Fatalf("entry (%s, USDT) not found", dateStr(day1))
		}
		assertDecimalEqual(t, "volume", entry.Volume, "150.00")
		assertDecimalEqual(t, "volumeUSD", entry.VolumeUSD, "150.00")
	})

	t.Run("multiple wagers same day + currency — summed into one bucket", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", day1)
		insertTx(t, "Wager", user, "ETH", "0.50000000", "1500.00", day1)
		insertTx(t, "Wager", user, "ETH", "0.25000000", "750.00", day1)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1 (all same day+currency)", len(got))
		}
		entry, ok := findDailyEntry(got, dateStr(day1), "ETH")
		if !ok {
			t.Fatalf("entry (%s, ETH) not found", dateStr(day1))
		}
		assertDecimalEqual(t, "volume (native sum)", entry.Volume, "1.75000000")
		assertDecimalEqual(t, "volumeUSD (sum)", entry.VolumeUSD, "5250.00")
	})

	t.Run("single day, multiple currencies — one row per currency", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", day1)
		insertTx(t, "Wager", user, "USDT", "500.00", "500.00", day1)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len: got %d, want 2 (ETH + USDT)", len(got))
		}
		eth, ok := findDailyEntry(got, dateStr(day1), "ETH")
		if !ok {
			t.Fatal("ETH entry not found")
		}
		assertDecimalEqual(t, "ETH volumeUSD", eth.VolumeUSD, "3000.00")

		usdt, ok := findDailyEntry(got, dateStr(day1), "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		assertDecimalEqual(t, "USDT volume", usdt.Volume, "500.00")
	})

	t.Run("multiple days — rows sorted by date ascending", func(t *testing.T) {
		clearCollection(t)
		// Insert out of order to confirm the pipeline sorts correctly.
		insertTx(t, "Wager", user, "USDT", "300.00", "300.00", day3)
		insertTx(t, "Wager", user, "USDT", "100.00", "100.00", day1)
		insertTx(t, "Wager", user, "USDT", "200.00", "200.00", day2)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len: got %d, want 3", len(got))
		}
		wantDates := []string{dateStr(day1), dateStr(day2), dateStr(day3)}
		for i, want := range wantDates {
			if got[i].Date != want {
				t.Errorf("row[%d].Date: got %q, want %q (not sorted by date asc)", i, got[i].Date, want)
			}
		}
	})

	t.Run("payout transactions are not included", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "100.00", "100.00", day1)
		insertTx(t, "Payout", user, "USDT", "9999.00", "9999.00", day1) // must be excluded

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1 (payout must not create an extra row)", len(got))
		}
		entry, ok := findDailyEntry(got, dateStr(day1), "USDT")
		if !ok {
			t.Fatal("USDT entry not found")
		}
		// Volume should reflect only the wager, not the payout.
		assertDecimalEqual(t, "volume (wager only)", entry.Volume, "100.00")
	})

	t.Run("date range: from filter — older wagers excluded", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "9999.00", "9999.00", day1) // before from
		insertTx(t, "Wager", user, "USDT", "200.00", "200.00", day3)   // after from

		got, err := testRepo.GetDailyWagerVolume(ctx, repository.DailyWagerVolumeFilter{From: &day2})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1 (day1 wager must be excluded)", len(got))
		}
		if got[0].Date != dateStr(day3) {
			t.Errorf("date: got %q, want %q", got[0].Date, dateStr(day3))
		}
		assertDecimalEqual(t, "volume", got[0].Volume, "200.00")
	})

	t.Run("date range: from+to filter — only in-window wagers counted", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "USDT", "9999.00", "9999.00", day1) // before from
		insertTx(t, "Wager", user, "USDT", "200.00", "200.00", day2)   // in window
		insertTx(t, "Wager", user, "USDT", "9999.00", "9999.00", day3) // after to

		got, err := testRepo.GetDailyWagerVolume(ctx, repository.DailyWagerVolumeFilter{From: &day2, To: &day2})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len: got %d, want 1 (only day2 in window)", len(got))
		}
		assertDecimalEqual(t, "volume", got[0].Volume, "200.00")
	})

	t.Run("multiple days + currencies — correct cross-product of buckets", func(t *testing.T) {
		clearCollection(t)
		insertTx(t, "Wager", user, "ETH", "1.00000000", "3000.00", day1)
		insertTx(t, "Wager", user, "USDT", "500.00", "500.00", day1)
		insertTx(t, "Wager", user, "ETH", "2.00000000", "6000.00", day2)
		insertTx(t, "Wager", user, "USDT", "800.00", "800.00", day2)

		got, err := testRepo.GetDailyWagerVolume(ctx, noFilter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 2 days × 2 currencies = 4 buckets
		if len(got) != 4 {
			t.Fatalf("len: got %d, want 4", len(got))
		}

		d1eth, ok := findDailyEntry(got, dateStr(day1), "ETH")
		if !ok {
			t.Fatal("(day1, ETH) not found")
		}
		assertDecimalEqual(t, "day1 ETH volumeUSD", d1eth.VolumeUSD, "3000.00")

		d2eth, ok := findDailyEntry(got, dateStr(day2), "ETH")
		if !ok {
			t.Fatal("(day2, ETH) not found")
		}
		assertDecimalEqual(t, "day2 ETH volumeUSD", d2eth.VolumeUSD, "6000.00")

		d1usdt, ok := findDailyEntry(got, dateStr(day1), "USDT")
		if !ok {
			t.Fatal("(day1, USDT) not found")
		}
		assertDecimalEqual(t, "day1 USDT volume", d1usdt.Volume, "500.00")

		d2usdt, ok := findDailyEntry(got, dateStr(day2), "USDT")
		if !ok {
			t.Fatal("(day2, USDT) not found")
		}
		assertDecimalEqual(t, "day2 USDT volume", d2usdt.Volume, "800.00")
	})
}
