package api_test

import (
	"admin-stats/repository"
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestGetGGRHandler(t *testing.T) {
	t.Run("missing auth returns 401", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/gross_gaming_rev", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("empty result returns 200 with empty data", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return nil, nil
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/gross_gaming_rev", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Data []any `json:"data"`
		}
		decodeJSON(t, resp, &body)
		if len(body.Data) != 0 {
			t.Errorf("data len: got %d, want 0", len(body.Data))
		}
	})

	t.Run("from after to returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/gross_gaming_rev?from=2024-06-01T00:00:00Z&to=2024-01-01T00:00:00Z", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)

		var body struct {
			Code string `json:"code"`
		}
		decodeJSON(t, resp, &body)
		if body.Code != "VALIDATION_ERROR" {
			t.Errorf("code: got %q, want VALIDATION_ERROR", body.Code)
		}
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return nil, errors.New("aggregation failed")
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/gross_gaming_rev", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusInternalServerError)
	})

	t.Run("result has correct currency field shape", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getGGRFn: func(_ context.Context, _ repository.GGRFilter) ([]repository.CurrencyTotals, error) {
				return []repository.CurrencyTotals{
					{Currency: "USDT", Wagers: "1000.00", Payouts: "600.00", WagersUSD: "1000.00", PayoutsUSD: "600.00"},
				}, nil
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/gross_gaming_rev", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Data []struct {
				Currency string `json:"currency"`
				GGRUSD   string `json:"ggrUSD"`
			} `json:"data"`
		}
		decodeJSON(t, resp, &body)
		if len(body.Data) != 1 {
			t.Fatalf("data len: got %d, want 1", len(body.Data))
		}
		if body.Data[0].Currency != "USDT" {
			t.Errorf("currency: got %q, want USDT", body.Data[0].Currency)
		}
		if body.Data[0].GGRUSD != "400.00" {
			t.Errorf("ggrUSD: got %q, want 400.00", body.Data[0].GGRUSD)
		}
	})
}

func TestGetDailyWagerVolumeHandler(t *testing.T) {
	t.Run("missing auth returns 401", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/daily_wager_volume", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("empty result returns 200", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return nil, nil
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/daily_wager_volume", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("from after to returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/daily_wager_volume?from=2024-06-01T00:00:00Z&to=2024-01-01T00:00:00Z", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getDailyWagerVolumeFn: func(_ context.Context, _ repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
				return nil, errors.New("aggregation failed")
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/daily_wager_volume", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusInternalServerError)
	})
}
