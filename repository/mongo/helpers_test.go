//go:build integration

package mongo_test

import (
	"admin-stats/model"
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

// insertTx is the base helper. insertWager and insertPayout delegate here so
// the only difference between them is the Type field — same pattern as the
// production service layer delegating to thin repo methods.
func insertTx(t *testing.T, txType string, userID bson.ObjectID, usdAmount string, at time.Time) {
	t.Helper()
	tx := model.Transaction{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		RoundID:   "round-test",
		Type:      txType,
		Amount:    decimal128("10.00"),
		Currency:  "USDT",
		USDAmount: decimal128(usdAmount),
		CreatedAt: at,
	}
	if err := testRepo.CreateTransaction(context.Background(), tx); err != nil {
		t.Fatalf("insertTx(%s): %v", txType, err)
	}
}

// insertWager inserts a single Wager transaction for the given user.
// Currency and native amount are fixed — aggregation pipelines rank on usdAmount only.
func insertWager(t *testing.T, userID bson.ObjectID, usdAmount string, at time.Time) {
	t.Helper()
	insertTx(t, "Wager", userID, usdAmount, at)
}

// insertPayout inserts a single Payout transaction.
// Use it in tests that verify the pipeline correctly ignores non-Wager types.
func insertPayout(t *testing.T, userID bson.ObjectID, usdAmount string, at time.Time) {
	t.Helper()
	insertTx(t, "Payout", userID, usdAmount, at)
}
