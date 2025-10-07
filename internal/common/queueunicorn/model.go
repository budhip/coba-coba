package queueunicorn

import (
	"golang.org/x/exp/maps"
)

const (
	HttpRequestJobKey = "HttpRequestJob"
)

type (
	RequestJobHTTP struct {
		Name    string     `json:"name"`
		Payload PayloadJob `json:"payload"`
		Options Options    `json:"options"`
	}

	PayloadJob struct {
		Host    string                 `json:"host"`
		Method  string                 `json:"method"`
		Body    interface{}            `json:"body"`
		Headers map[string]interface{} `json:"headers"`
		Tag     string                 `json:"tag,omitempty"`
	}

	Meta struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	Options struct {
		ProcessAt int `json:"process_at,omitempty"`
		MaxRetry  int `json:"max_retry,omitempty"`
		Timeout   int `json:"timeout,omitempty"`
		Deadline  int `json:"deadline,omitempty"`
	}

	ResponseSendMessage struct {
		Status   int      `json:"status"`
		Code     int      `json:"code"`
		Response Response `json:"response"`
		HttpCode int
	}

	Response struct {
		Meta
		Data  interface{} `json:"data"`
		Error string      `json:"error"`
	}
)

func RequestHeaderJob(secretKey, requestId string, additionalHeaders map[string]interface{}) map[string]interface{} {
	dst := map[string]interface{}{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Cache-Control": "no-cache",
		"User-Agent":    "go-queue-unicorn",
		"X-Secret-Key":  secretKey,
		"X-Request-Id":  requestId,
		"Is-Retry":      "true",
	}

	maps.Copy(dst, additionalHeaders)

	return dst
}
