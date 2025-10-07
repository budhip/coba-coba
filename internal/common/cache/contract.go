package cache

import (
	"context"
	"errors"
	"time"
)

type Client[T any] interface {
	Get(ctx context.Context, key string) (T, error)
	Set(ctx context.Context, key string, object T, ttl time.Duration) error
	GetOrSet(ctx context.Context, opts GetOrSetOpts[T]) (T, error)
}

var (
	ErrNotExists           = errors.New("key not exists on cache storage")
	ErrCallbackNotProvided = errors.New("callback not provided")
	ErrInvalidType         = errors.New("invalid type result")
)

type GetOrSetOpts[T any] struct {
	Key      string
	TTL      time.Duration
	Callback func() (T, error)
}
