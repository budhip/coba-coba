package models

import "time"

type Category struct {
	ID          int
	Code        string
	Name        string
	Description string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func (c *Category) ConvertToCategoryOut() *CategoryOut {
	return &CategoryOut{
		Kind:        "category",
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

type CreateCategoryIn struct {
	Code        string
	Name        string
	Description string
}

type CategoryOut struct {
	Kind        string     `json:"kind"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}

type CreateCategoryRequest struct {
	Code        string `json:"code" validate:"required,min=3,max=3,numeric"`
	Name        string `json:"name" validate:"required,min=1,max=50,nospecial,noStartEndSpaces"`
	Description string `json:"description" validate:"max=50"`
}
