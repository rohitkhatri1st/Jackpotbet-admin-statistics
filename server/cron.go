package server

import (
	"admin-stats/server/logger"
	"admin-stats/service"
	"context"
	"time"
)

// Cron runs the nightly stats recompute job.
// On startup it runs immediately to populate daily_stats, then fires every night at midnight UTC.
type Cron struct {
	stats         *service.StatsService
	recomputeDays int
	log           logger.Logger
}

func NewCron(stats *service.StatsService, recomputeDays int, log logger.Logger) *Cron {
	if recomputeDays <= 0 {
		recomputeDays = 7
	}
	return &Cron{stats: stats, recomputeDays: recomputeDays, log: log}
}

// Start launches the cron goroutine. It respects ctx cancellation for clean shutdown.
func (c *Cron) Start(ctx context.Context) {
	go func() {
		c.run(ctx)

		timer := time.NewTimer(untilNextMidnightUTC())
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			c.run(ctx)
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Cron) run(ctx context.Context) {
	now := time.Now().UTC()
	from := dayBoundary(now.AddDate(0, 0, -c.recomputeDays), true)
	to := dayBoundary(now.AddDate(0, 0, -1), false) // up to end of yesterday

	if err := c.stats.Recompute(ctx, from, to); err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info("msg", "daily stats recomputed",
		"from", from.Format("2006-01-02"),
		"to", to.Format("2006-01-02"),
	)
}

func untilNextMidnightUTC() time.Duration {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return next.Sub(now)
}

func dayBoundary(t time.Time, start bool) time.Time {
	if start {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
}
