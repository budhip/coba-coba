package safeaccess

import (
	"sync"
)

type Value[T any] struct {
	guard sync.RWMutex
	data  T
}

func New[T any](data T) *Value[T] {
	return &Value[T]{
		data: data,
	}
}

func (v *Value[T]) Load() T {
	v.guard.RLock()
	item := v.data
	v.guard.RUnlock()

	return item
}

func (v *Value[T]) Store(data T) {
	v.guard.Lock()
	v.data = data
	v.guard.Unlock()
}
