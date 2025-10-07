package models

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	IdempotencyStatusProcessFinished = "finished"
	IdempotencyStatusProcessPending  = "pending"

	TTLIdempotency = 1 * 24 * time.Hour // 1 day
)

type Idempotency struct {
	CacheKey string `json:"cacheKey"`

	StatusProcess string `json:"status"`

	// Fingerprints are used to identify the uniqueness of a request
	Fingerprint     string            `json:"fingerprint"`
	HTTPStatusCode  int               `json:"httpStatusCode"`
	ResponseBody    string            `json:"responseBody"`
	ResponseHeaders map[string]string `json:"responseHeaders"`
}

func NewIdempotency(key, status string, requestBody []byte) *Idempotency {
	fingerprint := sha1.Sum(requestBody)

	return &Idempotency{
		CacheKey:      fmt.Sprintf("locking-FP-%s", key),
		StatusProcess: status,
		Fingerprint:   hex.EncodeToString(fingerprint[:]),
	}
}

func (i *Idempotency) SetResponse(httpStatusCode int, responseHeaders map[string]string, responseBody string) {
	i.HTTPStatusCode = httpStatusCode
	i.ResponseHeaders = responseHeaders
	i.ResponseBody = responseBody
	i.StatusProcess = IdempotencyStatusProcessFinished
}
