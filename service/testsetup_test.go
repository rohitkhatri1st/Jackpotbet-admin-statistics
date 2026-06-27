package service_test

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"admin-stats/service"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── Mock repository ───────────────────────────────────────────────────────

// mockRepo implements repository.TransactionRepository via function fields.
//
// Why hand-written over mockery?
//   - No tooling dependency: go test ./... works without a code-generation step
//     or a CLI install in CI. mockery adds a .mockery.yaml and a re-gen requirement
//     whenever the interface changes.
//   - Closure captures feel natural: capturing a local variable (e.g. capturedLimit)
//     directly in the function literal is cleaner than asserting on mock.On(...) call
//     logs after the fact.
//   - Panics on unexpected calls: an unconfigured method field panics immediately
//     instead of silently returning zero values, making a missing mock setup
//     impossible to miss.
//
// When mockery would be worth it: if the interface grew beyond ~10 methods, changed
// frequently, or needed to be shared across multiple packages — auto-generation wins
// there. At 6 methods with a stable interface, the boilerplate cost is low.
type mockRepo struct {
	getTransactionsFn     func(context.Context, repository.TransactionFilter) ([]model.Transaction, error)
	createTransactionFn   func(context.Context, model.Transaction) error
	bulkInsertFn          func(context.Context, []model.Transaction) error
	getGGRFn              func(context.Context, repository.GGRFilter) ([]repository.CurrencyTotals, error)
	getDailyWagerVolumeFn func(context.Context, repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error)
	getUserWagerRankFn    func(context.Context, bson.ObjectID, repository.WagerRankFilter) (repository.UserWagerRank, error)
}

func (m *mockRepo) GetTransactions(ctx context.Context, f repository.TransactionFilter) ([]model.Transaction, error) {
	if m.getTransactionsFn == nil {
		panic("mockRepo.GetTransactions called but not configured")
	}
	return m.getTransactionsFn(ctx, f)
}

func (m *mockRepo) CreateTransaction(ctx context.Context, t model.Transaction) error {
	if m.createTransactionFn == nil {
		panic("mockRepo.CreateTransaction called but not configured")
	}
	return m.createTransactionFn(ctx, t)
}

func (m *mockRepo) BulkInsertTransactions(ctx context.Context, txs []model.Transaction) error {
	if m.bulkInsertFn == nil {
		panic("mockRepo.BulkInsertTransactions called but not configured")
	}
	return m.bulkInsertFn(ctx, txs)
}

func (m *mockRepo) GetGGR(ctx context.Context, f repository.GGRFilter) ([]repository.CurrencyTotals, error) {
	if m.getGGRFn == nil {
		panic("mockRepo.GetGGR called but not configured")
	}
	return m.getGGRFn(ctx, f)
}

func (m *mockRepo) GetDailyWagerVolume(ctx context.Context, f repository.DailyWagerVolumeFilter) ([]repository.DailyWagerVolumeEntry, error) {
	if m.getDailyWagerVolumeFn == nil {
		panic("mockRepo.GetDailyWagerVolume called but not configured")
	}
	return m.getDailyWagerVolumeFn(ctx, f)
}

func (m *mockRepo) GetUserWagerRank(ctx context.Context, id bson.ObjectID, f repository.WagerRankFilter) (repository.UserWagerRank, error) {
	if m.getUserWagerRankFn == nil {
		panic("mockRepo.GetUserWagerRank called but not configured")
	}
	return m.getUserWagerRankFn(ctx, id, f)
}

// compile-time check
var _ repository.TransactionRepository = (*mockRepo)(nil)

// ── No-op logger ──────────────────────────────────────────────────────────

type noopLogger struct{}

func (noopLogger) Debug(...any)              {}
func (noopLogger) Info(...any)               {}
func (noopLogger) Warn(...any)               {}
func (noopLogger) Error(error, ...any)       {}
func (noopLogger) With(...any) logger.Logger { return noopLogger{} }

// ── Service constructor helper ────────────────────────────────────────────

// newService builds a TransactionService wired to the given repo mock.
// Redis is nil so caching is skipped — tests focus on business logic only.
func newService(repo repository.TransactionRepository) *service.TransactionService {
	return service.NewTransactionService(&service.TransactionServiceOptions{
		Repo:  repo,
		Log:   noopLogger{},
		Redis: nil,
	})
}
