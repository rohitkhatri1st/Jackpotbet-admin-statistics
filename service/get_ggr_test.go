package service_test

import (
	"admin-stats/repository"
	"admin-stats/service"
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func assertDecimal(t *testing.T, label, got, want string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	if err != nil {
		t.Errorf("%s: cannot parse %q: %v", label, got, err)
		return
	}
	w, _ := decimal.NewFromString(want)
	if !g.Equal(w) {
		t.Errorf("%s: got %s, want %s", label, got, want)
	}
}

func TestGetGGR(t *testing.T) {
	ctx := context.Background()

	t.Run("nil input returns error", func(t *testing.T) {
		svc := newService(&mockRepo{})
		_, err := svc.GetGGR(ctx, nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})

	t.Run("GGR computed correctly — wagers minus payouts, native and USD", func(t *testing.T) {
		svc := newService(&mockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return []repository.CurrencyTotals{
					{
						Currency:   "USDT",
						Wagers:     "1000.00",
						Payouts:    "700.00",
						WagersUSD:  "1000.00",
						PayoutsUSD: "700.00",
					},
				}, nil
			},
		})

		got, err := svc.GetGGR(ctx, &service.GetGGRInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 1 {
			t.Fatalf("len: got %d, want 1", len(got.Data))
		}

		e := got.Data[0]
		assertDecimal(t, "GGR (native)", e.GGR, "300.00")
		assertDecimal(t, "GGRUSD", e.GGRUSD, "300.00")
		assertDecimal(t, "wagers", e.Wagers, "1000.00")
		assertDecimal(t, "payouts", e.Payouts, "700.00")
	})

	t.Run("negative GGR when payouts exceed wagers", func(t *testing.T) {
		svc := newService(&mockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return []repository.CurrencyTotals{
					{Currency: "USDT", Wagers: "100.00", Payouts: "150.00", WagersUSD: "100.00", PayoutsUSD: "150.00"},
				}, nil
			},
		})

		got, err := svc.GetGGR(ctx, &service.GetGGRInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "GGRUSD", got.Data[0].GGRUSD, "-50.00")
	})

	t.Run("empty payouts field treated as zero — GGR equals wagers", func(t *testing.T) {
		// The repo returns "" for Payouts when no payout transactions existed.
		// decimal.NewFromString("") returns an error; the service ignores it,
		// leaving payouts = 0. GGR should equal the wager amount.
		svc := newService(&mockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return []repository.CurrencyTotals{
					{Currency: "USDT", Wagers: "500.00", WagersUSD: "500.00"},
				}, nil
			},
		})

		got, err := svc.GetGGR(ctx, &service.GetGGRInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "GGR (no payouts)", got.Data[0].GGRUSD, "500.00")
	})

	t.Run("multiple currencies produce independent entries", func(t *testing.T) {
		svc := newService(&mockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return []repository.CurrencyTotals{
					{Currency: "ETH", Wagers: "2.00000000", Payouts: "1.00000000", WagersUSD: "6000.00", PayoutsUSD: "3000.00"},
					{Currency: "USDT", Wagers: "800.00", Payouts: "200.00", WagersUSD: "800.00", PayoutsUSD: "200.00"},
				}, nil
			},
		})

		got, err := svc.GetGGR(ctx, &service.GetGGRInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 2 {
			t.Fatalf("len: got %d, want 2", len(got.Data))
		}

		byCode := make(map[string]service.GGREntry)
		for _, e := range got.Data {
			byCode[e.Currency] = e
		}

		assertDecimal(t, "ETH GGR (native)", byCode["ETH"].GGR, "1.00000000")
		assertDecimal(t, "ETH GGRUSD", byCode["ETH"].GGRUSD, "3000.00")
		assertDecimal(t, "USDT GGRUSD", byCode["USDT"].GGRUSD, "600.00")
	})

	t.Run("repo error is propagated", func(t *testing.T) {
		repoErr := errors.New("aggregation failed")
		svc := newService(&mockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return nil, repoErr
			},
		})

		_, err := svc.GetGGR(ctx, &service.GetGGRInput{})
		if !errors.Is(err, repoErr) {
			t.Errorf("error: got %v, want %v", err, repoErr)
		}
	})
}
