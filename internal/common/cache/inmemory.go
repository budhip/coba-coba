package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type InMemoryClient[T any] struct {
	cache sync.Map
	done  chan struct{}
}

type cachedValue struct {
	Value string
	ExpAt time.Time
}

func (cv *cachedValue) expired() bool {
	return !cv.ExpAt.IsZero() && cv.ExpAt.Before(time.Now())
}

func NewInMemoryClient[T any]() *InMemoryClient[T] {
	m := &InMemoryClient[T]{
		done: make(chan struct{}),
	}

	go m.backgroundCleaner()
	return m
}

func (m *InMemoryClient[T]) Get(ctx context.Context, key string) (result T, err error) {
	valInterface, found := m.cache.Load(key)
	if !found {
		return result, ErrNotExists
	}

	val, ok := valInterface.(*cachedValue)
	if !ok {
		return result, ErrInvalidType
	}

	if val.expired() {
		m.cache.Delete(key)
		return result, ErrNotExists
	}

	if err = json.Unmarshal([]byte(val.Value), &result); err != nil {
		return result, err
	}

	return result, nil
}

func (m *InMemoryClient[T]) Set(ctx context.Context, key string, object T, ttl time.Duration) error {
	val, err := json.Marshal(object)
	if err != nil {
		return err
	}

	cv := &cachedValue{
		Value: string(val),
		ExpAt: time.Now().Add(ttl),
	}

	m.cache.Store(key, cv)
	return nil
}

func (m *InMemoryClient[T]) GetOrSet(ctx context.Context, opts GetOrSetOpts[T]) (result T, err error) {
	if opts.Callback == nil {
		return result, ErrCallbackNotProvided
	}

	obj, err := m.Get(ctx, opts.Key)
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

	// Set the new value in cache with the given TTL
	if err = m.Set(ctx, opts.Key, obj, opts.TTL); err != nil {
		return result, err
	}

	return obj, nil
}

// backgroundCleaner periodically removes expired entries from the cache.
func (m *InMemoryClient[T]) backgroundCleaner() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cache.Range(func(key, value interface{}) bool {
				cv, ok := value.(*cachedValue)
				if !ok || cv.expired() {
					m.cache.Delete(key)
				}
				return true // continue iteration
			})
		case <-m.done:
			return
		}
	}
}

// Close stops the background cleaner and releases resources.
func (m *InMemoryClient[T]) Close() {
	close(m.done)
}
