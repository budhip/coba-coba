package localstorage

import (
	"fmt"
	"os"
	"path"

	"github.com/dgraph-io/badger/v4"
)

type badgerStorage[T any] struct {
	db     *badger.DB
	bucket string
	pathDB string
}

func NewBadgerStorage[T any](bucket string) (LocalStorage[T], error) {
	pathDB := path.Join(os.TempDir(), bucket)

	opts := badger.DefaultOptions(pathDB)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &badgerStorage[T]{
		db:     db,
		bucket: bucket,
		pathDB: pathDB,
	}, nil
}

func (b badgerStorage[T]) Get(key string) (T, error) {
	var val T
	var rawVal []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		rawVal, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return val, fmt.Errorf("failed to get value from localstorage: %w", err)
	}

	if rawVal == nil {
		return val, nil
	}

	err = Unmarshal(rawVal, &val)
	if err != nil {
		return val, fmt.Errorf("failed to unmarshal value from localstorage: %w", err)
	}

	return val, nil
}

func (b badgerStorage[T]) Set(key string, value T) error {
	rawVal, err := Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), rawVal)
	})
	if err != nil {
		return fmt.Errorf("failed to set value to localstorage: %w", err)
	}

	return nil
}

func (b badgerStorage[T]) Delete(key string) error {
	err := b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	if err != nil {
		return fmt.Errorf("failed to delete value from localstorage: %w", err)
	}

	return nil
}

func (b badgerStorage[T]) Clean() error {
	return os.RemoveAll(b.pathDB)
}

func (b badgerStorage[T]) Close() error {
	return b.db.Close()
}

func (b badgerStorage[T]) ForEach(f func(key string, value T) error) error {
	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			var val T
			err = Unmarshal(v, &val)
			if err != nil {
				return fmt.Errorf("failed to unmarshal value: %w", err)
			}

			err = f(string(k), val)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate over localstorage: %w", err)
	}

	return nil
}
