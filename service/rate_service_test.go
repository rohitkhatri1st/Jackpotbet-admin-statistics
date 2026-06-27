package service_test

import (
	"admin-stats/service"
	"context"
	"testing"
)

func TestStaticRateService(t *testing.T) {
	ctx := context.Background()
	svc := service.NewStaticRateService()

	t.Run("USDT is 1:1 with USD", func(t *testing.T) {
		got, err := svc.ToUSD(ctx, "USDT", "250.00")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "USDT 250", got, "250.00")
	})

	t.Run("ETH at $3000", func(t *testing.T) {
		got, err := svc.ToUSD(ctx, "ETH", "2.00000000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "2 ETH", got, "6000.00")
	})

	t.Run("BTC at $65000", func(t *testing.T) {
		got, err := svc.ToUSD(ctx, "BTC", "0.50000000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "0.5 BTC", got, "32500.00")
	})

	t.Run("fractional ETH — decimal precision preserved", func(t *testing.T) {
		got, err := svc.ToUSD(ctx, "ETH", "1.33333333")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 1.33333333 × 3000 = 3999.99999 → rounds up to 4000.00 at 2dp
		assertDecimal(t, "1.33333333 ETH", got, "4000.00")
	})

	t.Run("zero amount", func(t *testing.T) {
		got, err := svc.ToUSD(ctx, "USDT", "0.00")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimal(t, "zero", got, "0.00")
	})

	t.Run("unsupported currency returns error", func(t *testing.T) {
		_, err := svc.ToUSD(ctx, "DOGE", "100.00")
		if err == nil {
			t.Error("expected error for unsupported currency DOGE")
		}
	})

	t.Run("invalid amount string returns error", func(t *testing.T) {
		_, err := svc.ToUSD(ctx, "USDT", "not-a-number")
		if err == nil {
			t.Error("expected error for non-numeric amount")
		}
	})

	t.Run("empty amount string returns error", func(t *testing.T) {
		_, err := svc.ToUSD(ctx, "USDT", "")
		if err == nil {
			t.Error("expected error for empty amount")
		}
	})
}
