package schema_test

import (
	"admin-stats/schema"
	"testing"
	"time"
)

func TestDateRangeFilter_Validate(t *testing.T) {
	t.Run("both nil — always valid", func(t *testing.T) {
		f := schema.DateRangeFilter{}
		if err := f.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("only From — normalised to UTC", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/New_York")
		ny := time.Date(2024, 3, 15, 10, 0, 0, 0, loc)
		f := schema.DateRangeFilter{From: &ny}
		if err := f.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.From.Location() != time.UTC {
			t.Errorf("From.Location: got %v, want UTC", f.From.Location())
		}
		if !f.From.Equal(ny.UTC()) {
			t.Errorf("From value changed after UTC conversion: got %v, want %v", f.From, ny.UTC())
		}
	})

	t.Run("only To — normalised to UTC", func(t *testing.T) {
		loc, _ := time.LoadLocation("Asia/Tokyo")
		tokyo := time.Date(2024, 6, 1, 9, 0, 0, 0, loc)
		f := schema.DateRangeFilter{To: &tokyo}
		if err := f.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.To.Location() != time.UTC {
			t.Errorf("To.Location: got %v, want UTC", f.To.Location())
		}
	})

	t.Run("From before To — valid", func(t *testing.T) {
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		f := schema.DateRangeFilter{From: &from, To: &to}
		if err := f.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("From equal To — invalid", func(t *testing.T) {
		ts := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		f := schema.DateRangeFilter{From: &ts, To: &ts}
		if err := f.Validate(); err == nil {
			t.Error("expected error when From == To")
		}
	})

	t.Run("From after To — invalid", func(t *testing.T) {
		from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		f := schema.DateRangeFilter{From: &from, To: &to}
		if err := f.Validate(); err == nil {
			t.Error("expected error when From is after To")
		}
	})

	t.Run("non-UTC From and To both normalised before comparison", func(t *testing.T) {
		// From in UTC+9 (Tokyo): 2024-01-02 01:00 local = 2024-01-01 16:00 UTC
		// To in UTC-5 (NY):     2024-01-01 13:00 local = 2024-01-01 18:00 UTC
		// After normalisation: From (16:00 UTC) < To (18:00 UTC) — valid.
		tokyo, _ := time.LoadLocation("Asia/Tokyo")
		ny, _ := time.LoadLocation("America/New_York")
		from := time.Date(2024, 1, 2, 1, 0, 0, 0, tokyo) // 2024-01-01 16:00 UTC
		to := time.Date(2024, 1, 1, 13, 0, 0, 0, ny)     // 2024-01-01 18:00 UTC
		f := schema.DateRangeFilter{From: &from, To: &to}
		if err := f.Validate(); err != nil {
			t.Errorf("unexpected error after timezone normalisation: %v", err)
		}
	})

	t.Run("zero-value pointer treated as unset — no error", func(t *testing.T) {
		// The qs library initialises *time.Time fields to &time.Time{} (zero value)
		// when the query key is absent rather than leaving them nil.
		// Validate() must treat a zero-value pointer the same as nil so that
		// requests with no date filter are accepted, not rejected.
		zero := time.Time{}
		f := schema.DateRangeFilter{From: &zero, To: &zero}
		if err := f.Validate(); err != nil {
			t.Errorf("zero-value pointers should be treated as unset, got error: %v", err)
		}
		if f.From != nil {
			t.Error("From should be reset to nil after being detected as zero")
		}
		if f.To != nil {
			t.Error("To should be reset to nil after being detected as zero")
		}
	})

	t.Run("Validate mutates From and To to UTC in-place", func(t *testing.T) {
		loc, _ := time.LoadLocation("Europe/Paris")
		from := time.Date(2024, 5, 1, 12, 0, 0, 0, loc)
		to := time.Date(2024, 5, 31, 12, 0, 0, 0, loc)
		f := schema.DateRangeFilter{From: &from, To: &to}
		_ = f.Validate()
		if f.From.Location() != time.UTC {
			t.Error("From was not mutated to UTC in-place")
		}
		if f.To.Location() != time.UTC {
			t.Error("To was not mutated to UTC in-place")
		}
	})
}
