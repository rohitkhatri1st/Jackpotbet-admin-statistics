package service

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type TransactionService struct {
	repo        repository.TransactionRepository
	log         logger.Logger
	rateService RateService
	// Simple Redis key-based caching for stat endpoints (GGR, daily wager volume).
	// Each unique (from, to) date pair is stored as a JSON blob with a configurable TTL.
	//
	// A better approach for production would be a materialized view pattern:
	//   - A `daily_stats` MongoDB collection stores pre-computed wager/payout totals
	//     per (date, currency), populated nightly by a cron job.
	//   - GGR sums `daily_stats` instead of scanning raw transactions (O(days) not O(millions)).
	//   - Daily wager volume reads from `daily_stats` directly.
	//   - Redis caches those reads — only today's bucket ever changes, making invalidation trivial.
	//
	// However, the materialized view introduces non-trivial complexity: the cron must handle
	// DB corrections (e.g. backdated transactions inserted manually), cache invalidation must
	// be coordinated across the nightly recompute and any manual correction flow, and the
	// recompute window (how many past days to redo) needs to be tuned carefully. For the scope
	// of this assignment, simple TTL-based caching is used instead.
	redis    *redis.Client
	cacheTTL time.Duration
}

type TransactionServiceOptions struct {
	Repo     repository.TransactionRepository
	Log      logger.Logger
	Redis    *redis.Client
	CacheTTL time.Duration // defaults to 24h if zero
}

func NewTransactionService(opts *TransactionServiceOptions) *TransactionService {
	ttl := opts.CacheTTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &TransactionService{
		repo:        opts.Repo,
		log:         opts.Log,
		rateService: NewStaticRateService(),
		redis:       opts.Redis,
		cacheTTL:    ttl,
	}
}

type CreateTransactionInput struct {
	UserID    bson.ObjectID
	RoundID   string
	Type      string
	Currency  string
	Amount    string // pre-validated decimal128 string
	CreatedAt *time.Time
}

type GetTransactionsInput struct {
	From   *time.Time
	To     *time.Time
	Cursor *bson.ObjectID
	Limit  int64
}

type TransactionsResult struct {
	Data   []model.Transaction `json:"data"`
	Cursor *bson.ObjectID      `json:"cursor"`
}

func (s *TransactionService) GetTransactions(ctx context.Context, input *GetTransactionsInput) (*TransactionsResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	// Fetch one extra to determine if a next page exists.
	transactions, err := s.repo.GetTransactions(ctx, repository.TransactionFilter{
		From:   input.From,
		To:     input.To,
		Cursor: input.Cursor,
		Limit:  input.Limit + 1,
	})
	if err != nil {
		return nil, err
	}

	var nextCursor *bson.ObjectID
	if int64(len(transactions)) > input.Limit {
		transactions = transactions[:input.Limit]
		id := transactions[len(transactions)-1].ID
		nextCursor = &id
	}

	return &TransactionsResult{
		Data:   transactions,
		Cursor: nextCursor,
	}, nil
}

func (s *TransactionService) CreateTransaction(ctx context.Context, input *CreateTransactionInput) (*model.Transaction, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	usdAmount, err := s.rateService.ToUSD(ctx, input.Currency, input.Amount)
	if err != nil {
		return nil, err
	}

	// ParseDecimal128 cannot fail — caller must validate the format beforehand.
	amount, _ := bson.ParseDecimal128(input.Amount)
	usd, _ := bson.ParseDecimal128(usdAmount)

	createdAt := time.Now().UTC()
	if input.CreatedAt != nil {
		createdAt = input.CreatedAt.UTC()
	}

	t := &model.Transaction{
		ID:        bson.NewObjectID(),
		CreatedAt: createdAt,
		UserID:    input.UserID,
		RoundID:   input.RoundID,
		Type:      input.Type,
		Amount:    amount,
		Currency:  input.Currency,
		USDAmount: usd,
	}

	if err := s.repo.CreateTransaction(ctx, *t); err != nil {
		return nil, err
	}
	return t, nil
}

type GetDailyWagerVolumeInput struct {
	From *time.Time
	To   *time.Time
}

type CurrencyVolume struct {
	Volume    string `json:"volume"`
	VolumeUSD string `json:"volumeUSD"`
}

type DailyVolumeEntry struct {
	Date           string                    `json:"date"`
	TotalVolumeUSD string                    `json:"totalVolumeUSD"`
	Currencies     map[string]CurrencyVolume `json:"currencies"`
}

type DailyWagerVolumeResult struct {
	Data []DailyVolumeEntry `json:"data"`
}

func (s *TransactionService) GetDailyWagerVolume(ctx context.Context, input *GetDailyWagerVolumeInput) (*DailyWagerVolumeResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	if s.redis != nil {
		key := statsCacheKey("daily_wager", input.From, input.To)
		if cached, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var result DailyWagerVolumeResult
			if json.Unmarshal(cached, &result) == nil {
				return &result, nil
			}
		}
	}

	rows, err := s.repo.GetDailyWagerVolume(ctx, repository.DailyWagerVolumeFilter{
		From: input.From,
		To:   input.To,
	})
	if err != nil {
		return nil, err
	}

	result := groupDailyWagersByDate(rows)

	if s.redis != nil {
		key := statsCacheKey("daily_wager", input.From, input.To)
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, key, data, s.cacheTTL)
		}
	}

	return result, nil
}

// groupDailyWagersByDate converts flat (date, currency) repo rows into one
// DailyVolumeEntry per date, preserving the sort order returned by the repository.
func groupDailyWagersByDate(rows []repository.DailyWagerVolumeEntry) *DailyWagerVolumeResult {
	dateOrder := make([]string, 0)
	totalUSDByDate := make(map[string]decimal.Decimal)
	// map of [date -> [map of currency -> volume]]
	currenciesByDate := make(map[string]map[string]CurrencyVolume)

	for _, row := range rows {
		if _, exists := currenciesByDate[row.Date]; !exists {
			currenciesByDate[row.Date] = make(map[string]CurrencyVolume)
			dateOrder = append(dateOrder, row.Date)
		}
		volume, _ := decimal.NewFromString(row.Volume)
		volumeUSD, _ := decimal.NewFromString(row.VolumeUSD)
		currenciesByDate[row.Date][row.Currency] = CurrencyVolume{
			Volume:    volume.StringFixed(8),
			VolumeUSD: volumeUSD.StringFixed(2),
		}
		totalUSDByDate[row.Date] = totalUSDByDate[row.Date].Add(volumeUSD)
	}

	result := make([]DailyVolumeEntry, 0, len(dateOrder))
	for _, date := range dateOrder {
		result = append(result, DailyVolumeEntry{
			Date:           date,
			TotalVolumeUSD: totalUSDByDate[date].StringFixed(2),
			Currencies:     currenciesByDate[date],
		})
	}

	return &DailyWagerVolumeResult{Data: result}
}

type GetGGRInput struct {
	From *time.Time
	To   *time.Time
}

type GGREntry struct {
	Currency   string `json:"currency"`
	Wagers     string `json:"wagers"`
	Payouts    string `json:"payouts"`
	GGR        string `json:"ggr"`
	WagersUSD  string `json:"wagersUSD"`
	PayoutsUSD string `json:"payoutsUSD"`
	GGRUSD     string `json:"ggrUSD"`
}

type GGRResult struct {
	Data []GGREntry `json:"data"`
}

func (s *TransactionService) GetGGR(ctx context.Context, input *GetGGRInput) (*GGRResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	if s.redis != nil {
		key := statsCacheKey("ggr", input.From, input.To)
		if cached, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var result GGRResult
			if json.Unmarshal(cached, &result) == nil {
				return &result, nil
			}
		}
	}

	totals, err := s.repo.GetGGR(ctx, repository.GGRFilter{
		From: input.From,
		To:   input.To,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]GGREntry, 0, len(totals))
	for _, t := range totals {
		// Convert string amounts to decimal.Decimal for arithmetic, then back to string with fixed precision.
		// Errors are ignored because the repository guarantees valid decimal strings.
		wagers, _ := decimal.NewFromString(t.Wagers)
		payouts, _ := decimal.NewFromString(t.Payouts)
		wagersUSD, _ := decimal.NewFromString(t.WagersUSD)
		payoutsUSD, _ := decimal.NewFromString(t.PayoutsUSD)

		entries = append(entries, GGREntry{
			Currency:   t.Currency,
			Wagers:     wagers.StringFixed(8),
			Payouts:    payouts.StringFixed(8),
			GGR:        wagers.Sub(payouts).StringFixed(8),
			WagersUSD:  wagersUSD.StringFixed(2),
			PayoutsUSD: payoutsUSD.StringFixed(2),
			GGRUSD:     wagersUSD.Sub(payoutsUSD).StringFixed(2),
		})
	}

	result := &GGRResult{Data: entries}

	if s.redis != nil {
		key := statsCacheKey("ggr", input.From, input.To)
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, key, data, s.cacheTTL)
		}
	}

	return result, nil
}

type GetWagerPercentileInput struct {
	UserID bson.ObjectID
	From   *time.Time
	To     *time.Time
}

type WagerPercentileResult struct {
	UserID        string `json:"userId"`
	TotalUSD      string `json:"totalUSD"`
	Rank          int    `json:"rank"`
	TotalUsers    int    `json:"totalUsers"`
	TopPercentile string `json:"topPercentile"` // rank/totalUsers*100, e.g. "2.00" means top 2%
}

func (s *TransactionService) GetWagerPercentile(ctx context.Context, input *GetWagerPercentileInput) (*WagerPercentileResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	if s.redis != nil {
		key := userStatsCacheKey("wager_percentile", input.UserID.Hex(), input.From, input.To)
		if cached, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var result WagerPercentileResult
			if json.Unmarshal(cached, &result) == nil {
				return &result, nil
			}
		}
	}

	rank, err := s.repo.GetUserWagerRank(ctx, input.UserID, repository.WagerRankFilter{
		From: input.From,
		To:   input.To,
	})
	if err != nil {
		return nil, err
	}

	userRank := rank.Rank
	totalUsers := rank.TotalUsers
	totalUSD := decimal.Zero

	if rank.Found {
		totalUSD, _ = decimal.NewFromString(rank.TotalUSD)
	} else {
		// User had no wagers in the period — place them last.
		totalUsers++
		userRank = totalUsers
	}

	topPercentile := decimal.NewFromInt(int64(userRank)).
		Div(decimal.NewFromInt(int64(totalUsers))).
		Mul(decimal.NewFromInt(100)).
		StringFixed(2)

	result := &WagerPercentileResult{
		UserID:        input.UserID.Hex(),
		TotalUSD:      totalUSD.StringFixed(2),
		Rank:          userRank,
		TotalUsers:    totalUsers,
		TopPercentile: topPercentile,
	}

	if s.redis != nil {
		key := userStatsCacheKey("wager_percentile", input.UserID.Hex(), input.From, input.To)
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, key, data, s.cacheTTL)
		}
	}

	return result, nil
}

// statsCacheKey builds a Redis key for a global stat result (GGR, daily wager volume).
// Dates are normalised to UTC day strings so queries spanning the same calendar days
// share a cache entry regardless of the exact time-of-day component in the filter.
func statsCacheKey(prefix string, from, to *time.Time) string {
	fromStr, toStr := "all", "all"
	if from != nil {
		fromStr = from.UTC().Format("2006-01-02")
	}
	if to != nil {
		toStr = to.UTC().Format("2006-01-02")
	}
	return fmt.Sprintf("%s:%s:%s", prefix, fromStr, toStr)
}

// userStatsCacheKey is like statsCacheKey but scoped to a specific user.
func userStatsCacheKey(prefix, userID string, from, to *time.Time) string {
	return fmt.Sprintf("%s:%s", userID, statsCacheKey(prefix, from, to))
}
