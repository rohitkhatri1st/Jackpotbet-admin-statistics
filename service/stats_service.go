package service

import (
	"admin-stats/model"
	"admin-stats/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

const defaultStatsCacheTTL = 24 * time.Hour

type StatsService struct {
	txRepo    repository.TransactionRepository
	statsRepo repository.DailyStatsRepository
	redis     *redis.Client
	cacheTTL  time.Duration
}

type StatsServiceOptions struct {
	TxRepo    repository.TransactionRepository
	StatsRepo repository.DailyStatsRepository
	Redis     *redis.Client
	CacheTTL  time.Duration // 0 = default 24h
}

func NewStatsService(opts *StatsServiceOptions) *StatsService {
	ttl := opts.CacheTTL
	if ttl == 0 {
		ttl = defaultStatsCacheTTL
	}
	return &StatsService{
		txRepo:    opts.TxRepo,
		statsRepo: opts.StatsRepo,
		redis:     opts.Redis,
		cacheTTL:  ttl,
	}
}

// Recompute re-aggregates raw transactions for [from, to], upserts the results into
// daily_stats, and flushes all Redis stat cache entries so the next read is fresh.
func (s *StatsService) Recompute(ctx context.Context, from, to time.Time) error {
	stats, err := s.txRepo.ComputeDailyStats(ctx, from, to)
	if err != nil {
		return err
	}
	if err := s.statsRepo.Upsert(ctx, stats); err != nil {
		return err
	}
	if s.redis != nil {
		s.flushStatsCache(ctx)
	}
	return nil
}

// flushStatsCache deletes all cached GGR and daily wager volume keys from Redis.
// The keyspace is tiny (one key per unique date-range query), so SCAN is acceptable.
func (s *StatsService) flushStatsCache(ctx context.Context) {
	for _, pattern := range []string{"ggr:*", "daily_wager:*"} {
		var cursor uint64
		for {
			keys, next, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			if len(keys) > 0 {
				s.redis.Del(ctx, keys...)
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
}

// ---------- GGR -----------------------------------------------------------------

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

func (s *StatsService) GetGGR(ctx context.Context, input *GetGGRInput) (*GGRResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}
	isHistorical := input.To != nil && input.To.Before(startOfDay(time.Now().UTC()))

	if isHistorical && s.redis != nil {
		key := statsCacheKey("ggr", input.From, input.To)
		if data, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var cached GGRResult
			if json.Unmarshal(data, &cached) == nil {
				return &cached, nil
			}
		}
	}

	stats, err := s.getStatsForRange(ctx, input.From, input.To)
	if err != nil {
		return nil, err
	}
	result := buildGGRResult(stats)

	if isHistorical && s.redis != nil {
		key := statsCacheKey("ggr", input.From, input.To)
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, key, data, s.cacheTTL)
		}
	}
	return result, nil
}

func buildGGRResult(stats []model.DailyStats) *GGRResult {
	type totals struct{ wagers, payouts, wagersUSD, payoutsUSD decimal.Decimal }
	byC := make(map[string]*totals)

	for _, s := range stats {
		t := byC[s.Currency]
		if t == nil {
			t = &totals{}
			byC[s.Currency] = t
		}
		w, _ := decimal.NewFromString(s.Wagers.String())
		p, _ := decimal.NewFromString(s.Payouts.String())
		wUSD, _ := decimal.NewFromString(s.WagersUSD.String())
		pUSD, _ := decimal.NewFromString(s.PayoutsUSD.String())
		t.wagers = t.wagers.Add(w)
		t.payouts = t.payouts.Add(p)
		t.wagersUSD = t.wagersUSD.Add(wUSD)
		t.payoutsUSD = t.payoutsUSD.Add(pUSD)
	}

	entries := make([]GGREntry, 0, len(byC))
	for currency, t := range byC {
		entries = append(entries, GGREntry{
			Currency:   currency,
			Wagers:     t.wagers.StringFixed(8),
			Payouts:    t.payouts.StringFixed(8),
			GGR:        t.wagers.Sub(t.payouts).StringFixed(8),
			WagersUSD:  t.wagersUSD.StringFixed(2),
			PayoutsUSD: t.payoutsUSD.StringFixed(2),
			GGRUSD:     t.wagersUSD.Sub(t.payoutsUSD).StringFixed(2),
		})
	}
	return &GGRResult{Data: entries}
}

// ---------- Daily Wager Volume --------------------------------------------------

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

func (s *StatsService) GetDailyWagerVolume(ctx context.Context, input *GetDailyWagerVolumeInput) (*DailyWagerVolumeResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}
	isHistorical := input.To != nil && input.To.Before(startOfDay(time.Now().UTC()))

	if isHistorical && s.redis != nil {
		key := statsCacheKey("daily_wager", input.From, input.To)
		if data, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var cached DailyWagerVolumeResult
			if json.Unmarshal(data, &cached) == nil {
				return &cached, nil
			}
		}
	}

	stats, err := s.getStatsForRange(ctx, input.From, input.To)
	if err != nil {
		return nil, err
	}
	result := buildDailyWagerVolumeResult(stats)

	if isHistorical && s.redis != nil {
		key := statsCacheKey("daily_wager", input.From, input.To)
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, key, data, s.cacheTTL)
		}
	}
	return result, nil
}

// buildDailyWagerVolumeResult groups daily stats into one entry per date,
// preserving the sort order returned by the repository.
func buildDailyWagerVolumeResult(stats []model.DailyStats) *DailyWagerVolumeResult {
	dateOrder := make([]string, 0)
	totalUSDByDate := make(map[string]decimal.Decimal)
	currenciesByDate := make(map[string]map[string]CurrencyVolume)

	for _, s := range stats {
		if _, exists := currenciesByDate[s.Date]; !exists {
			currenciesByDate[s.Date] = make(map[string]CurrencyVolume)
			dateOrder = append(dateOrder, s.Date)
		}
		volume, _ := decimal.NewFromString(s.Wagers.String())
		volumeUSD, _ := decimal.NewFromString(s.WagersUSD.String())
		currenciesByDate[s.Date][s.Currency] = CurrencyVolume{
			Volume:    volume.StringFixed(8),
			VolumeUSD: volumeUSD.StringFixed(2),
		}
		totalUSDByDate[s.Date] = totalUSDByDate[s.Date].Add(volumeUSD)
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

// ---------- Shared helpers ------------------------------------------------------

// getStatsForRange returns daily stats from the pre-computed collection for historical
// ranges, and falls back to computing from raw transactions for live ranges (to >= today).
func (s *StatsService) getStatsForRange(ctx context.Context, from, to *time.Time) ([]model.DailyStats, error) {
	if to != nil && to.Before(startOfDay(time.Now().UTC())) {
		return s.statsRepo.GetByDateRange(ctx, repository.DailyStatsFilter{From: from, To: to})
	}
	f := time.Time{}
	if from != nil {
		f = *from
	}
	t := time.Now().UTC()
	if to != nil {
		t = *to
	}
	return s.txRepo.ComputeDailyStats(ctx, f, t)
}

func statsCacheKey(prefix string, from, to *time.Time) string {
	f := "0"
	if from != nil {
		f = strconv.FormatInt(startOfDay(*from).Unix(), 10)
	}
	t := "0"
	if to != nil {
		t = strconv.FormatInt(endOfDay(*to).Unix(), 10)
	}
	return fmt.Sprintf("%s:%s:%s", prefix, f, t)
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func endOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
}
