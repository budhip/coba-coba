package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisClient[T any] struct {
	redis *redis.Client
}

func NewRedisClient[T any](redis *redis.Client) Client[T] {
	return &redisClient[T]{redis: redis}
}

func (r redisClient[T]) Get(ctx context.Context, key string) (result T, err error) {
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return result, ErrNotExists
		}
		return result, err
	}

	if err = json.Unmarshal([]byte(val), &result); err != nil {
		return result, err
	}

	return result, nil
}

func (r redisClient[T]) Set(ctx context.Context, key string, object T, ttl time.Duration) error {
	val, err := json.Marshal(object)
	if err != nil {
		return err
	}

	return r.redis.Set(ctx, key, val, ttl).Err()
}

func (r redisClient[T]) GetOrSet(ctx context.Context, opts GetOrSetOpts[T]) (result T, err error) {
	if opts.Callback == nil {
		return result, ErrCallbackNotProvided
	}

	obj, err := r.Get(ctx, opts.Key)
	if err == nil {
		return obj, nil
	}

	if err != ErrNotExists {
		return result, err
	}

	// Key does not exist, call the callback to get the new value
	obj, err = opts.Callback()
	if err != nil {
		return result, err
	}

	// Set the new value in Redis with the given TTL
	if err = r.Set(ctx, opts.Key, obj, opts.TTL); err != nil {
		return result, err
	}

	return obj, nil
}
