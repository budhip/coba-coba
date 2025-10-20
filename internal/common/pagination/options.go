package pagination

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"fmt"
)

// Options contains pagination parameters
type Options struct {
	Limit      int
	NextCursor string
	PrevCursor string
}

// BuildCursorAndLimit builds cursor and limit with validation
func (o *Options) BuildCursorAndLimit() (*BaseCursor, int, error) {
	limit := o.Limit

	// Set default limit
	if limit == 0 {
		limit = constants.DefaultLimit
	}

	// Validate limit
	if limit < 0 {
		return nil, 0, fmt.Errorf("the limit must be greater than zero")
	}

	// Use over-fetch limit to check if next page exists
	limit += constants.OverFetchOffset

	// Build cursor
	cursor, err := o.buildCursor()
	if err != nil {
		return nil, 0, err
	}

	return cursor, limit, nil
}

func (o *Options) buildCursor() (*BaseCursor, error) {
	// Forward pagination
	if o.NextCursor != "" {
		cursor, err := DecodeCursor(o.NextCursor)
		if err != nil {
			return nil, err
		}
		return cursor, nil
	}

	// Backward pagination
	if o.PrevCursor != "" {
		cursor, err := DecodeCursor(o.PrevCursor)
		if err != nil {
			return nil, err
		}
		cursor.SetBackward(true)
		return cursor, nil
	}

	return nil, nil
}
