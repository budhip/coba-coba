package models

import (
	"fmt"
)

func GetCacheKeyStatusRetryDLQ(processRetryId string) string {
	return fmt.Sprintf("dlq:go-fp-transaction:status-retry:%s", processRetryId)
}

type StatusRetryDLQ struct {
	// ProcessId is a unique id for each process that will be retried
	ProcessId string `json:"processId"`

	// ProcessName is a name of process that will be retried
	// currently only support "account stream" and "order transaction"
	ProcessName string `json:"processName"`

	// MaxRetry is a maximum retry that will be done
	MaxRetry int `json:"maxRetry"`

	// CurrentRetry is a current retry that has been done
	CurrentRetry int `json:"currentRetry"`
}

func (status StatusRetryDLQ) ToHeaders() map[string]any {
	return map[string]any{
		"X-DLQ-Process-Id":   status.ProcessId,
		"X-DLQ-Process-Type": status.ProcessName,
		"X-DLQ-Max-Retry":    fmt.Sprintf("%d", status.MaxRetry),
	}
}
