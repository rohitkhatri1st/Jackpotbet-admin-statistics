package service_test

import (
	"admin-stats/repository"
	"admin-stats/service"
	"context"
	"errors"
	"testing"
)

func TestGetDailyWagerVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("nil input returns error", func(t *testing.T) {
		svc := newService(&mockRepo{})
		_, err := svc.GetDailyWagerVolume(ctx, nil)
		if err == nil {
			t.Error("expected error for nil input")
		}
	})

	t.Run("flat repo rows are grouped into one entry per date", func(t *testing.T) {
		// Repo returns two (date, currency) rows for the same date.
		// The service must group them into a single DailyVolumeEntry.
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return []repository.DailyWagerVolumeEntry{
					{Date: "2024-01-10", Currency: "ETH", Volume: "1.00000000", VolumeUSD: "3000.00"},
					{Date: "2024-01-10", Currency: "USDT", Volume: "500.00", VolumeUSD: "500.00"},
				}, nil
			},
		})

		got, err := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 1 {
			t.Fatalf("len: got %d, want 1 (both rows share the same date)", len(got.Data))
		}

		entry := got.Data[0]
		if entry.Date != "2024-01-10" {
			t.Errorf("date: got %q, want 2024-01-10", entry.Date)
		}
		if _, ok := entry.Currencies["ETH"]; !ok {
			t.Error("ETH missing from currencies map")
		}
		if _, ok := entry.Currencies["USDT"]; !ok {
			t.Error("USDT missing from currencies map")
		}
	})

	t.Run("TotalVolumeUSD sums across all currencies for the day", func(t *testing.T) {
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return []repository.DailyWagerVolumeEntry{
					{Date: "2024-01-10", Currency: "ETH", Volume: "1.00000000", VolumeUSD: "3000.00"},
					{Date: "2024-01-10", Currency: "USDT", Volume: "500.00", VolumeUSD: "500.00"},
				}, nil
			},
		})

		got, _ := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})

		assertDecimal(t, "TotalVolumeUSD", got.Data[0].TotalVolumeUSD, "3500.00")
	})

	t.Run("date order from repo is preserved", func(t *testing.T) {
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return []repository.DailyWagerVolumeEntry{
					{Date: "2024-01-10", Currency: "USDT", Volume: "100.00", VolumeUSD: "100.00"},
					{Date: "2024-01-11", Currency: "USDT", Volume: "200.00", VolumeUSD: "200.00"},
					{Date: "2024-01-12", Currency: "USDT", Volume: "300.00", VolumeUSD: "300.00"},
				}, nil
			},
		})

		got, err := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 3 {
			t.Fatalf("len: got %d, want 3", len(got.Data))
		}

		wantDates := []string{"2024-01-10", "2024-01-11", "2024-01-12"}
		for i, want := range wantDates {
			if got.Data[i].Date != want {
				t.Errorf("row[%d].Date: got %q, want %q (order not preserved)", i, got.Data[i].Date, want)
			}
		}
	})

	t.Run("multiple days with multiple currencies — correct cross-product grouping", func(t *testing.T) {
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return []repository.DailyWagerVolumeEntry{
					{Date: "2024-01-10", Currency: "ETH", Volume: "1.00000000", VolumeUSD: "3000.00"},
					{Date: "2024-01-10", Currency: "USDT", Volume: "200.00", VolumeUSD: "200.00"},
					{Date: "2024-01-11", Currency: "ETH", Volume: "2.00000000", VolumeUSD: "6000.00"},
					{Date: "2024-01-11", Currency: "USDT", Volume: "400.00", VolumeUSD: "400.00"},
				}, nil
			},
		})

		got, err := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 2 {
			t.Fatalf("len: got %d, want 2 (one entry per date)", len(got.Data))
		}

		assertDecimal(t, "day1 TotalVolumeUSD", got.Data[0].TotalVolumeUSD, "3200.00")
		assertDecimal(t, "day2 TotalVolumeUSD", got.Data[1].TotalVolumeUSD, "6400.00")
	})

	t.Run("empty repo result returns empty data slice", func(t *testing.T) {
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return nil, nil
			},
		})

		got, err := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Data) != 0 {
			t.Errorf("len: got %d, want 0", len(got.Data))
		}
	})

	t.Run("repo error is propagated", func(t *testing.T) {
		repoErr := errors.New("aggregation failed")
		svc := newService(&mockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return nil, repoErr
			},
		})

		_, err := svc.GetDailyWagerVolume(ctx, &service.GetDailyWagerVolumeInput{})
		if !errors.Is(err, repoErr) {
			t.Errorf("error: got %v, want %v", err, repoErr)
		}
	})
}
