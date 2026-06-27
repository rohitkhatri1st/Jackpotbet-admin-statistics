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

func (f DateRangeFilter) Validate() error {
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		return errors.New("'from' must be before 'to'")
	}
	return nil
}
