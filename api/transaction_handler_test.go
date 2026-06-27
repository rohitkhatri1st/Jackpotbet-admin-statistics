package api_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetTransactionsHandler(t *testing.T) {
	t.Run("missing auth token returns 401", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("empty result returns 200 with empty data array", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return nil, nil
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Data   []any          `json:"data"`
			Cursor *bson.ObjectID `json:"cursor"`
		}
		decodeJSON(t, resp, &body)
		if len(body.Data) != 0 {
			t.Errorf("data len: got %d, want 0", len(body.Data))
		}
		if body.Cursor != nil {
			t.Error("cursor: got non-nil, want nil")
		}
	})

	t.Run("limit above 100 returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions?limit=200", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid cursor hex returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions?cursor=not-a-hex-id", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("from after to returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		// time.Time.UnmarshalText (used by qs) expects RFC 3339 format.
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions?from=2024-02-01T00:00:00Z&to=2024-01-01T00:00:00Z", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getTransactionsFn: func(_ context.Context, _ repository.TransactionFilter) ([]model.Transaction, error) {
				return nil, errors.New("db unavailable")
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/transactions", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusInternalServerError)

		var body struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		decodeJSON(t, resp, &body)
		if body.Code != "INTERNAL_ERROR" {
			t.Errorf("code: got %q, want INTERNAL_ERROR", body.Code)
		}
	})
}

func TestCreateTransactionHandler(t *testing.T) {
	validBody := func() []byte {
		return []byte(`{
			"userId":   "507f1f77bcf86cd799439011",
			"roundId":  "round-1",
			"type":     "Wager",
			"currency": "USDT",
			"amount":   "100.00"
		}`)
	}

	t.Run("missing auth token returns 401", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader(validBody()))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("valid request returns 201 with transaction body", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			createTransactionFn: func(_ context.Context, _ model.Transaction) error { return nil },
		})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader(validBody()))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		var body map[string]any
		decodeJSON(t, resp, &body)
		if body["currency"] != "USDT" {
			t.Errorf("currency: got %v, want USDT", body["currency"])
		}
	})

	t.Run("missing required field returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		body := []byte(`{"roundId":"r1","type":"Wager","currency":"USDT","amount":"10.00"}`) // userId missing
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid type value returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		body := []byte(`{"userId":"507f1f77bcf86cd799439011","roundId":"r1","type":"Bet","currency":"USDT","amount":"10.00"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader([]byte(`{bad json`)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("empty body returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("wrong Content-Type returns 415", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/transactions", bytes.NewReader(validBody()))
		req.Header.Set("Content-Type", "text/plain")
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnsupportedMediaType)
	})
}
