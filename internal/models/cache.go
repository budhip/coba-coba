package models

import (
	"time"
)

type GetOrSetCacheOptions[T any] struct {
	Key string
	TTL time.Duration
	Fn  func() (T, error)
}

type SetCacheOptions[T any] struct {
	Key string
	TTL time.Duration
	Val T
}
