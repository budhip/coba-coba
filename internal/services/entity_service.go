package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type EntityService interface {
	Create(ctx context.Context, req models.CreateEntityIn) (out *models.Entity, err error)
	GetAll(ctx context.Context) (out *[]models.Entity, err error)
}

type entity service

var _ EntityService = (*entity)(nil)

// Create implements EntityService.
func (s *entity) Create(ctx context.Context, req models.CreateEntityIn) (out *models.Entity, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check exist
	exist, err := s.srv.sqlRepo.GetEntityRepository().GetByCode(ctx, req.Code)
	if err != nil {
		err = common.ErrUnableToCreate
		return
	}
	if exist != nil {
		err = common.ErrDataExist
		return
	}

	// Insert
	out, err = s.srv.sqlRepo.GetEntityRepository().Create(ctx, &req)
	if err != nil {
		return
	}

	return
}

// GetAll implements EntityService.
func (s *entity) GetAll(ctx context.Context) (out *[]models.Entity, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Get data
	out, err = s.srv.sqlRepo.GetEntityRepository().List(ctx)
	if err != nil {
		err = common.ErrInternalServerError
		return
	}

	return
}
