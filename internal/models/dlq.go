package models

import (
	"time"
)

type FailedMessage struct {
	Payload    []byte    `json:"payload"`
	Timestamp  time.Time `json:"timestamp"`
	CauseError error     `json:"-"`

	// Error is a string representation of CauseError
	Error string `json:"error"`
}
