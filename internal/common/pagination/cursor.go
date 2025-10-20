package pagination

import (
	"encoding/base64"
	"fmt"
)

// Cursor interface for generic cursor operations
type Cursor interface {
	GetID() string
	IsBackward() bool
	Encode() string
}

// BaseCursor provides common cursor implementation
type BaseCursor struct {
	ID         string
	isBackward bool
}

func NewBaseCursor(id string, isBackward bool) *BaseCursor {
	return &BaseCursor{
		ID:         id,
		isBackward: isBackward,
	}
}

func (c *BaseCursor) GetID() string {
	return c.ID
}

func (c *BaseCursor) IsBackward() bool {
	return c.isBackward
}

func (c *BaseCursor) SetBackward(backward bool) {
	c.isBackward = backward
}

func (c *BaseCursor) Encode() string {
	return base64.StdEncoding.EncodeToString([]byte(c.ID))
}

func (c *BaseCursor) String() string {
	return c.Encode()
}

// DecodeCursor decodes base64 encoded cursor string
func DecodeCursor(cursor string) (*BaseCursor, error) {
	if cursor == "" {
		return nil, fmt.Errorf("cursor is empty")
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cursor string: %w", err)
	}

	id := string(decodedBytes)
	if id == "" {
		return nil, fmt.Errorf("failed to parse cursor string: invalid format")
	}

	return NewBaseCursor(id, false), nil
}
