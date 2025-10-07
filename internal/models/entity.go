package models

import (
	"time"
)

type Entity struct {
	ID          int
	Code        string
	Name        string
	Description string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

type EntityOut struct {
	Kind        string     `json:"kind"`
	ID          int        `json:"-"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"-"`
	UpdatedAt   *time.Time `json:"-"`
}

func (e *Entity) ToResponse() *EntityOut {
	return &EntityOut{
		Kind:        "entity",
		Code:        e.Code,
		Name:        e.Name,
		Description: e.Description,
	}
}

type CreateEntityIn struct {
	Code        string
	Name        string
	Description string
}

type CreateEntityRequest struct {
	Code        string `json:"code" validate:"required,numeric,min=3,max=3"`
	Name        string `json:"name" validate:"required,min=1,max=50,nospecial,noStartEndSpaces"`
	Description string `json:"description" validate:"max=50"`
}
