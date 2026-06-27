package api_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"admin-stats/server/middleware"
	"admin-stats/server/validator"
	"admin-stats/service"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/bson"

	"admin-stats/api"
)

const testToken = "test-internal-token"

// ── Mock repository ───────────────────────────────────────────────────────────

// apiMockRepo is the same function-field pattern used in service tests.
// Defined separately here so the api_test package has no import cycle on service_test.
type apiMockRepo struct {
	getTransactionsFn     func(context.Context, repository.TransactionFilter) ([]model.Transaction, error)
	createTransactionFn   func(context.Context, model.Transaction) error
	bulkInsertFn          func(context.Context, []model.Transaction) error
	getGGRFn              func(context.Context, repository.GGRFilter) ([]repository.CurrencyTotals, error)
	getDailyWagerVolumeFn func(context.Context, repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error)
	getUserWagerRankFn    func(context.Context, bson.ObjectID, repository.WagerRankFilter) (repository.UserWagerRank, error)
}

func (m *apiMockRepo) GetTransactions(ctx context.Context, f repository.TransactionFilter) ([]model.Transaction, error) {
	if m.getTransactionsFn == nil {
		panic("apiMockRepo.GetTransactions called but not configured")
	}
	return m.getTransactionsFn(ctx, f)
}
func (m *apiMockRepo) CreateTransaction(ctx context.Context, t model.Transaction) error {
	if m.createTransactionFn == nil {
		panic("apiMockRepo.CreateTransaction called but not configured")
	}
	return m.createTransactionFn(ctx, t)
}
func (m *apiMockRepo) BulkInsertTransactions(ctx context.Context, txs []model.Transaction) error {
	if m.bulkInsertFn == nil {
		panic("apiMockRepo.BulkInsertTransactions called but not configured")
	}
	return m.bulkInsertFn(ctx, txs)
}
func (m *apiMockRepo) GetGGR(ctx context.Context, f repository.GGRFilter) ([]repository.CurrencyTotals, error) {
	if m.getGGRFn == nil {
		panic("apiMockRepo.GetGGR called but not configured")
	}
	return m.getGGRFn(ctx, f)
}
func (m *apiMockRepo) GetDailyWagerVolume(ctx context.Context, f repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
	if m.getDailyWagerVolumeFn == nil {
		panic("apiMockRepo.GetDailyWagerVolume called but not configured")
	}
	return m.getDailyWagerVolumeFn(ctx, f)
}
func (m *apiMockRepo) GetUserWagerRank(ctx context.Context, id bson.ObjectID, f repository.WagerRankFilter) (repository.UserWagerRank, error) {
	if m.getUserWagerRankFn == nil {
		panic("apiMockRepo.GetUserWagerRank called but not configured")
	}
	return m.getUserWagerRankFn(ctx, id, f)
}

var _ repository.TransactionRepository = (*apiMockRepo)(nil)

// ── No-op logger ──────────────────────────────────────────────────────────────

type apiNoopLogger struct{}

func (apiNoopLogger) Debug(...any)              {}
func (apiNoopLogger) Info(...any)               {}
func (apiNoopLogger) Warn(...any)               {}
func (apiNoopLogger) Error(error, ...any)       {}
func (apiNoopLogger) With(...any) logger.Logger { return apiNoopLogger{} }

// ── Test API builder ──────────────────────────────────────────────────────────

// newTestServer wires up a full API server backed by the given mock repo.
// All internal routes are protected by a static token middleware using testToken.
// Returns the httptest.Server — caller must call srv.Close() via t.Cleanup.
func newTestServer(t *testing.T, repo repository.TransactionRepository) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	log := apiNoopLogger{}
	svc := service.NewTransactionService(&service.TransactionServiceOptions{
		Repo:  repo,
		Log:   log,
		Redis: nil,
	})
	a := api.New(&api.Options{
		Router:                 router,
		Log:                    log,
		ForceLog:               log,
		Validator:              validator.NewValidator(),
		Services:               &service.Services{Transaction: svc},
		InternalAuthMiddleware: middleware.InternalAuth(log, testToken),
	})
	_ = a
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// authHeader returns the X-Internal-Key header set to testToken.
func authHeader(req *http.Request) *http.Request {
	req.Header.Set("X-Internal-Key", testToken)
	return req
}

// decodeJSON unmarshals the response body into dst.
func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
}

// assertStatus fails the test if the response status code does not match want.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status: got %d, want %d", resp.StatusCode, want)
	}
}
