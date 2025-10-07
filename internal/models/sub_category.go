package models

import "time"

type SubCategory struct {
	ID           int
	CategoryCode string
	Code         string
	Name         string
	Description  string
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

type CreateSubCategory struct {
	CategoryCode string
	Code         string
	Name         string
	Description  string // Optional
}

type SubCategoryOut struct {
	Kind         string     `json:"kind"`
	ID           int        `json:"-"`
	CategoryCode string     `json:"categoryCode"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	CreatedAt    *time.Time `json:"-"`
	UpdatedAt    *time.Time `json:"-"`
}

func (c *SubCategory) ToResponse() *SubCategoryOut {
	return &SubCategoryOut{
		Kind:         "subCategory",
		CategoryCode: c.CategoryCode,
		Code:         c.Code,
		Name:         c.Name,
		Description:  c.Description,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

type CreateSubCategoryRequest struct {
	CategoryCode string `json:"categoryCode" validate:"required,numeric,min=3,max=3"`
	Code         string `json:"code" validate:"required,numeric,min=5,max=5"`
	Name         string `json:"name" validate:"required,min=1,max=50,nospecial,noStartEndSpaces"`
	Description  string `json:"description" validate:"max=50"`
}
