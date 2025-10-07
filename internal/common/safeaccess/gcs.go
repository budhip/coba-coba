package safeaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// ObjectStorageClient is a client that reads and writes a file from/to the object storage.
// it uses safeaccess to ensure that the file is read and written atomically.
type ObjectStorageClient[T any] interface {
	// LoadFile reads the file from the storage and unmarshal it into the local value.
	LoadFile(ctx context.Context) error

	// UpdateFile marshals the local value into the file storage.
	UpdateFile(ctx context.Context) error

	// Value returns the value.
	Value() *Value[T]
}

type GCSJson[T any] struct {
	object *storage.ObjectHandle
	val    Value[T]
}

func NewGCSJson[T any](object *storage.ObjectHandle) *GCSJson[T] {
	return &GCSJson[T]{
		object: object,
	}
}

func (g *GCSJson[T]) LoadFile(ctx context.Context) error {
	r, err := g.object.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}

		return fmt.Errorf("failed to init reader: %w", err)
	}
	defer r.Close()

	r.Attrs.CacheControl = "no-store"

	bFile, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var obj T
	err = json.Unmarshal(bFile, &obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	g.val.Store(obj)

	return nil
}

func (g *GCSJson[T]) UpdateFile(ctx context.Context) error {
	w := g.object.NewWriter(ctx)

	val := g.val.Load()
	bFile, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal file: %w", err)
	}

	w.ContentType = "application/json"
	w.CacheControl = "no-store"

	_, err = w.Write(bFile)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

func (g *GCSJson[T]) Value() *Value[T] {
	return &g.val
}
