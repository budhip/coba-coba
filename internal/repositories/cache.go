package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

type cacheClient struct {
	redis *redis.Client
}

func NewCacheRepository(redis *redis.Client) CacheRepository {
	return &cacheClient{redis: redis}
}

func (cc *cacheClient) SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return cc.redis.SetNX(ctx, key, value, ttl).Result()
}

func (cc *cacheClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return cc.redis.Set(ctx, key, value, ttl).Err()
}

func (cc *cacheClient) Get(ctx context.Context, key string) (string, error) {
	val, err := cc.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return val, common.ErrDataNotFound
		}
		return val, err
	}
	val = strings.TrimSpace(val)

	return val, nil
}

func (cc *cacheClient) Del(ctx context.Context, keys ...string) error {
	return cc.redis.Del(ctx, keys...).Err()
}
