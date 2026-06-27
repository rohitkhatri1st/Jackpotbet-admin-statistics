package api_test

import (
	"admin-stats/repository"
	"context"
	"errors"
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGetWagerPercentileHandler(t *testing.T) {
	validUserID := bson.NewObjectID().Hex()

	t.Run("missing auth returns 401", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/"+validUserID+"/wager_percentile", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("invalid userId hex returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/not-a-valid-id/wager_percentile", nil)
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

	t.Run("valid user returns 200 with percentile fields", func(t *testing.T) {
		targetID, _ := bson.ObjectIDFromHex(validUserID)
		srv := newTestServer(t, &apiMockRepo{
			getUserWagerRankFn: func(_ context.Context, id bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{Found: true, TotalUSD: "2500.00", Rank: 5, TotalUsers: 100}, nil
			},
		})
		_ = targetID
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/"+validUserID+"/wager_percentile", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Rank          int    `json:"rank"`
			TotalUsers    int    `json:"totalUsers"`
			TopPercentile string `json:"topPercentile"`
			TotalUSD      string `json:"totalUSD"`
		}
		decodeJSON(t, resp, &body)
		if body.Rank != 5 {
			t.Errorf("rank: got %d, want 5", body.Rank)
		}
		if body.TotalUsers != 100 {
			t.Errorf("totalUsers: got %d, want 100", body.TotalUsers)
		}
		if body.TopPercentile != "5.00" {
			t.Errorf("topPercentile: got %q, want 5.00 (5/100×100)", body.TopPercentile)
		}
	})

	t.Run("user not found — placed last, returns 200", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{Found: false, TotalUsers: 49}, nil
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/"+validUserID+"/wager_percentile", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var body struct {
			Rank          int    `json:"rank"`
			TotalUsers    int    `json:"totalUsers"`
			TopPercentile string `json:"topPercentile"`
		}
		decodeJSON(t, resp, &body)
		// 49 wagering users + the user themselves = 50; placed last = rank 50
		if body.Rank != 50 {
			t.Errorf("rank: got %d, want 50", body.Rank)
		}
		if body.TopPercentile != "100.00" {
			t.Errorf("topPercentile: got %q, want 100.00", body.TopPercentile)
		}
	})

	t.Run("from after to returns 400", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/"+validUserID+"/wager_percentile?from=2024-06-01T00:00:00Z&to=2024-01-01T00:00:00Z", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		srv := newTestServer(t, &apiMockRepo{
			getUserWagerRankFn: func(_ context.Context, _ bson.ObjectID, _ repository.WagerRankFilter) (repository.UserWagerRank, error) {
				return repository.UserWagerRank{}, errors.New("rank query failed")
			},
		})
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/user/"+validUserID+"/wager_percentile", nil)
		resp, err := http.DefaultClient.Do(authHeader(req))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusInternalServerError)
	})
}
