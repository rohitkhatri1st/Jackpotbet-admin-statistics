package schema

import (
	"errors"
	"time"
)

// DateRangeFilter is embedded in query structs that accept optional from/to date bounds.
// Call Validate() after decoding to enforce from < to.
type DateRangeFilter struct {
	From *time.Time `qs:"from"`
	To   *time.Time `qs:"to"`
}

// Validate normalises From/To to UTC (clients may send any timezone offset) and
// checks that From is before To when both are present.
func (f *DateRangeFilter) Validate() error {
	if f.From != nil {
		t := f.From.UTC()
		f.From = &t
	}
	if f.To != nil {
		t := f.To.UTC()
		f.To = &t
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		return errors.New("'from' must be before 'to'")
	}
	return nil
}
