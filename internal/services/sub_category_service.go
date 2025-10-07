package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type SubCategoryService interface {
	Create(ctx context.Context, req models.CreateSubCategory) (output *models.SubCategory, err error)
	GetAll(ctx context.Context) (out *[]models.SubCategory, err error)
}

type subCategory service

var _ SubCategoryService = (*subCategory)(nil)

// Create implements SubCategoryService.
func (s *subCategory) Create(ctx context.Context, req models.CreateSubCategory) (output *models.SubCategory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check category
	cat, err := s.srv.sqlRepo.GetCategoryRepository().GetByCode(ctx, req.CategoryCode)
	if err != nil {
		return
	}
	if cat == nil {
		err = common.ErrDataNotFound
		return
	}

	// Check subcategory
	subCat, err := s.srv.sqlRepo.GetSubCategoryRepository().GetByCode(ctx, req.Code)
	if err != nil {
		return
	}
	if subCat != nil {
		err = common.ErrDataExist
		return
	}

	// Insert
	output, err = s.srv.sqlRepo.GetSubCategoryRepository().Create(ctx, &req)
	if err != nil {
		err = common.ErrUnableToCreate
		return
	}

	return
}

// GetAll implements SubCategoryService.
func (s *subCategory) GetAll(ctx context.Context) (out *[]models.SubCategory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	out, err = s.srv.sqlRepo.GetSubCategoryRepository().GetAll(ctx)
	if err != nil {
		return
	}

	return
}
