// White-box test file (package service, not service_test) so we can reach
// the unexported statsCacheKey and userStatsCacheKey helpers.
package service

import (
	"testing"
	"time"
)

func TestStatsCacheKey(t *testing.T) {
	t.Run("both nil dates become 'all'", func(t *testing.T) {
		got := statsCacheKey("ggr", nil, nil)
		if got != "ggr:all:all" {
			t.Errorf("got %q, want ggr:all:all", got)
		}
	})

	t.Run("only from set", func(t *testing.T) {
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		got := statsCacheKey("ggr", &from, nil)
		if got != "ggr:2024-01-01:all" {
			t.Errorf("got %q, want ggr:2024-01-01:all", got)
		}
	})

	t.Run("only to set", func(t *testing.T) {
		to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		got := statsCacheKey("daily_wager", nil, &to)
		if got != "daily_wager:all:2024-12-31" {
			t.Errorf("got %q, want daily_wager:all:2024-12-31", got)
		}
	})

	t.Run("both dates set", func(t *testing.T) {
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		got := statsCacheKey("ggr", &from, &to)
		if got != "ggr:2024-01-01:2024-01-31" {
			t.Errorf("got %q, want ggr:2024-01-01:2024-01-31", got)
		}
	})

	t.Run("non-UTC date is normalised — same calendar day shares one key", func(t *testing.T) {
		// 2024-03-15 23:00 in UTC+2 = 2024-03-15 21:00 UTC → key date is "2024-03-15"
		loc, _ := time.LoadLocation("Europe/Helsinki") // UTC+2
		from := time.Date(2024, 3, 15, 23, 0, 0, 0, loc)
		gotKey := statsCacheKey("ggr", &from, nil)

		fromUTC := time.Date(2024, 3, 15, 21, 0, 0, 0, time.UTC) // same moment in UTC
		wantKey := statsCacheKey("ggr", &fromUTC, nil)

		if gotKey != wantKey {
			t.Errorf("non-UTC date produced different key: got %q, want %q", gotKey, wantKey)
		}
	})

	t.Run("time-of-day component is stripped — only date matters", func(t *testing.T) {
		morning := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
		evening := time.Date(2024, 6, 15, 22, 30, 0, 0, time.UTC)
		if statsCacheKey("ggr", &morning, nil) != statsCacheKey("ggr", &evening, nil) {
			t.Error("two times on the same UTC day should produce the same cache key")
		}
	})
}

func TestUserStatsCacheKey(t *testing.T) {
	userID := "507f1f77bcf86cd799439011"

	t.Run("key is prefixed with userID", func(t *testing.T) {
		got := userStatsCacheKey("wager_percentile", userID, nil, nil)
		want := "507f1f77bcf86cd799439011:wager_percentile:all:all"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("different users produce different keys for same date range", func(t *testing.T) {
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		key1 := userStatsCacheKey("wager_percentile", "aaabbbccc", &from, nil)
		key2 := userStatsCacheKey("wager_percentile", "dddeeefff", &from, nil)
		if key1 == key2 {
			t.Error("different users should produce different cache keys")
		}
	})

	t.Run("same user + same dates always produce the same key", func(t *testing.T) {
		from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
		k1 := userStatsCacheKey("wager_percentile", userID, &from, &to)
		k2 := userStatsCacheKey("wager_percentile", userID, &from, &to)
		if k1 != k2 {
			t.Errorf("identical inputs produced different keys: %q vs %q", k1, k2)
		}
	})
}
