package matcher

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/mock/gomock"
)

type deadlineRangeMatcher struct {
	min time.Duration
	max time.Duration
}

func (m deadlineRangeMatcher) Matches(x interface{}) bool {
	ctx, ok := x.(context.Context)
	if !ok {
		return false
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	remaining := time.Until(deadline)
	// require remaining to be positive and within the range
	return remaining > 0 && remaining >= m.min && remaining <= m.max
}

func (m deadlineRangeMatcher) String() string {
	return fmt.Sprintf("context with deadline in [%s, %s]", m.min, m.max)
}

func ContextWithTimeoutRange(min, max time.Duration) gomock.Matcher {
	return deadlineRangeMatcher{min: min, max: max}
}
