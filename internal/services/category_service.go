package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type CategoryService interface {
	Create(ctx context.Context, req models.CreateCategoryIn) (output *models.Category, err error)
	GetAll(ctx context.Context) (output *[]models.Category, err error)
}

type category service

var _ CategoryService = (*category)(nil)

// Create implements CategoryService.
func (s *category) Create(ctx context.Context, req models.CreateCategoryIn) (output *models.Category, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check existing
	exist, err := s.srv.sqlRepo.GetCategoryRepository().GetByCode(ctx, req.Code)
	if err != nil {
		err = common.ErrUnableToCreate
		return
	}
	if exist != nil {
		err = common.ErrDataExist
		return
	}

	// Insert
	res, err := s.srv.sqlRepo.GetCategoryRepository().Create(ctx, &req)
	if err != nil {
		err = common.ErrUnableToCreate
		return
	}
	output = res

	return
}

// GetAll implements CategoryService.
func (s *category) GetAll(ctx context.Context) (output *[]models.Category, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Get data
	categories, err := s.srv.sqlRepo.GetCategoryRepository().List(ctx)
	if err != nil {
		err = common.ErrInternalServerError
		return
	}

	output = categories

	return
}
