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

// Add batch formatting helper
func FormatTimeToRFC3339(t *time.Time) string {
	return FormatNullableTime(t, time.RFC3339)
}

func FormatTimesToRFC3339(times ...*time.Time) []string {
	return FormatNullableTimes(time.RFC3339, times...)
}
