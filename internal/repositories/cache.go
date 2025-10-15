package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

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

func GetOrSetCache[T any](ctx context.Context, repo CacheRepository, opts models.GetOrSetCacheOptions[T]) (T, error) {
	res, err := GetCache[T](ctx, repo, opts.Key)
	if err == nil {
		return res, nil
	}

	if !errors.Is(err, common.ErrDataNotFound) {
		return res, err
	}

	res, err = opts.Fn()
	if err != nil {
		return res, err
	}

	err = SetCache(ctx, repo, models.SetCacheOptions[T]{
		Key: opts.Key,
		TTL: opts.TTL,
		Val: res,
	})
	if err != nil {
		return res, err
	}

	return res, nil
}

func GetCache[T any](ctx context.Context, repo CacheRepository, key string) (T, error) {
	var res T
	jsonStr, err := repo.Get(ctx, key)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal([]byte(jsonStr), &res)
	return res, err
}

func SetCache[T any](ctx context.Context, repo CacheRepository, opts models.SetCacheOptions[T]) error {
	b, err := json.Marshal(opts.Val)
	if err != nil {
		return err
	}

	return repo.Set(ctx, opts.Key, string(b), opts.TTL)
}
