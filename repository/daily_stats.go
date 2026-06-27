package repository

import (
	"admin-stats/model"
	"context"
	"time"
)

type DailyStatsFilter struct {
	From *time.Time
	To   *time.Time
}

type DailyStatsRepository interface {
	EnsureIndexes(ctx context.Context) error
	Upsert(ctx context.Context, stats []model.DailyStats) error
	GetByDateRange(ctx context.Context, filter DailyStatsFilter) ([]model.DailyStats, error)
}
