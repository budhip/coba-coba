package dateutil

import (
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
)

// FormatNullableTime formats time pointer or returns default placeholder
func FormatNullableTime(t *time.Time, layout string) string {
	if t == nil {
		return constants.DefaultDatePlaceholder
	}
	if layout == "" {
		layout = time.RFC3339
	}
	return t.Format(layout)
}

// FormatNullableTimes formats multiple time pointers
func FormatNullableTimes(layout string, times ...*time.Time) []string {
	results := make([]string, len(times))
	for i, t := range times {
		results[i] = FormatNullableTime(t, layout)
	}
	return results
}
