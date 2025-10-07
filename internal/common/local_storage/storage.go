package localstorage

import (
	"encoding/json"
)

// LocalStorage is an interface to store data locally
// It can be used to store data in memory or in a file
// Main purpose is to store data for single process only, like reconciliation data
type LocalStorage[T any] interface {
	// Get retrieves data from localstorage
	// If key is not found, it will return empty value
	Get(key string) (T, error)

	// Set is used to store data to localstorage
	Set(key string, value T) error

	// Delete is used to delete data from localstorage
	Delete(key string) error

	// ForEach is used to iterate all data in storage
	ForEach(func(key string, value T) error) error

	// Close is used to close the storage
	Close() error

	// Clean is used to clean all data in storage
	Clean() error
}

type (
	// MarshalFunc define
	MarshalFunc func(v any) ([]byte, error)

	// UnmarshalFunc define
	UnmarshalFunc func(data []byte, v any) error
)

var (
	Marshal   MarshalFunc   = json.Marshal
	Unmarshal UnmarshalFunc = json.Unmarshal
)
