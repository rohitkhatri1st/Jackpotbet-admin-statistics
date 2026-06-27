//go:build integration

package mongo_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── Collection helpers ────────────────────────────────────────────────────

// clearCollection wipes all transaction documents. Call at the start of each
// sub-test so tests never see each other's data. testDB is used directly so we
// don't rely on any unexported field from the repository.
func clearCollection(t *testing.T) {
	t.Helper()
	_, err := testDB.Collection("transactions").DeleteMany(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("clearCollection: %v", err)
	}
}

// ── Decimal helpers ───────────────────────────────────────────────────────

// decimal128 parses a decimal string (e.g. "1500.00") into bson.Decimal128.
// Panics on an invalid string — only call with string literals in tests.
func decimal128(s string) bson.Decimal128 {
	d, err := bson.ParseDecimal128(s)
	if err != nil {
		panic("decimal128: invalid string " + s)
	}
	return d
}

// assertDecimalEqual fails the test if got and want do not represent the same
// decimal value. Comparison is value-based (ignores trailing zeros).
func assertDecimalEqual(t *testing.T, label, got, want string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	if err != nil {
		t.Errorf("%s: cannot parse got value %q: %v", label, got, err)
		return
	}
	w, err := decimal.NewFromString(want)
	if err != nil {
		t.Errorf("%s: cannot parse want value %q: %v", label, want, err)
		return
	}
	if !g.Equal(w) {
		t.Errorf("%s: got %s, want %s", label, got, want)
	}
}

// ── Transaction insert helpers ────────────────────────────────────────────

// insertTx is the base helper all typed wrappers delegate to.
// currency defaults to "USDT" and amount to "10.00" when passed as empty strings,
// which is sufficient for tests that only care about usdAmount (e.g. rank tests).
// Pass explicit values when the test needs to verify currency-specific aggregation
// (e.g. GGR, daily wager volume).
func insertTx(t *testing.T, txType string, userID bson.ObjectID, currency, amount, usdAmount string, at time.Time) {
	t.Helper()
	if currency == "" {
		currency = "USDT"
	}
	if amount == "" {
		amount = "10.00"
	}
	tx := model.Transaction{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		RoundID:   "round-test",
		Type:      txType,
		Amount:    decimal128(amount),
		Currency:  currency,
		USDAmount: decimal128(usdAmount),
		CreatedAt: at,
	}
	if err := testRepo.CreateTransaction(context.Background(), tx); err != nil {
		t.Fatalf("insertTx(%s): %v", txType, err)
	}
}

// insertWager inserts a Wager with default currency (USDT) and native amount.
// Sufficient for rank tests that only care about usdAmount.
func insertWager(t *testing.T, userID bson.ObjectID, usdAmount string, at time.Time) {
	t.Helper()
	insertTx(t, "Wager", userID, "", "", usdAmount, at)
}

// insertPayout inserts a Payout with default currency (USDT) and native amount.
func insertPayout(t *testing.T, userID bson.ObjectID, usdAmount string, at time.Time) {
	t.Helper()
	insertTx(t, "Payout", userID, "", "", usdAmount, at)
}

// ── Result lookup helpers ─────────────────────────────────────────────────

// findCurrency locates a CurrencyTotals entry by currency code in a slice.
// The GGR pipeline result order is non-deterministic (map iteration), so
// tests must look up by currency rather than by index.
func findCurrency(totals []repository.CurrencyTotals, currency string) (repository.CurrencyTotals, bool) {
	for _, t := range totals {
		if t.Currency == currency {
			return t, true
		}
	}
	return repository.CurrencyTotals{}, false
}

// findDailyEntry locates a DailyWagerVolumeEntry by (date, currency) pair.
// The pipeline sorts by (date, currency) so order is deterministic, but
// lookup by key is safer than by index when sub-tests insert varying data.
func findDailyEntry(entries []repository.DailyWagerVolumeEntry, date, currency string) (repository.DailyWagerVolumeEntry, bool) {
	for _, e := range entries {
		if e.Date == date && e.Currency == currency {
			return e, true
		}
	}
	return repository.DailyWagerVolumeEntry{}, false
}

// dateStr formats a time.Time as the "YYYY-MM-DD" string the pipeline produces
// via $dateToString. Use this whenever a test needs to assert on Date fields.
func dateStr(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
