package repositories

import (
	"context"
	"database/sql"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type EntityRepository interface {
	CheckEntityByCode(ctx context.Context, code string) (err error)
	Create(ctx context.Context, in *models.CreateEntityIn) (created *models.Entity, err error)
	GetByCode(ctx context.Context, code string) (*models.Entity, error)
	List(ctx context.Context) (*[]models.Entity, error)
}

type entityRepository sqlRepo

var _ EntityRepository = (*entityRepository)(nil)

func (r *entityRepository) CheckEntityByCode(ctx context.Context, code string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	var entityCode string
	err = db.QueryRowContext(ctx, queryEntityIsExistByCode, code).Scan(
		&entityCode,
	)
	if err != nil {
		return
	}

	return
}

// Create implements EntityRepository.
func (r *entityRepository) Create(ctx context.Context, in *models.CreateEntityIn) (created *models.Entity, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	in.Name = strings.ToUpper(in.Name)
	args, err := common.GetFieldValues(*in)
	if err != nil {
		return
	}

	var entity models.Entity
	err = db.QueryRowContext(ctx, queryEntityCreate, args...).Scan(
		&entity.ID,
		&entity.Code,
		&entity.Name,
		&entity.Description,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		err = common.ErrUnableToCreate
		return
	}
	created = &entity

	return
}

// GetByCode retrieves an Entity by its code.
func (r *entityRepository) GetByCode(ctx context.Context, code string) (*models.Entity, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	var entity models.Entity
	err = db.QueryRowContext(ctx, queryEntityGetByCode, code).Scan(
		&entity.ID,
		&entity.Description,
		&entity.Code,
		&entity.Name,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

// List implements EntityRepository.
func (r *entityRepository) List(ctx context.Context) (*[]models.Entity, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	// Execute the query with QueryContext
	rows, err := db.QueryContext(ctx, queryEntityList)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the result set and process the data
	var result []models.Entity
	for rows.Next() {
		var entity models.Entity
		if err := rows.Scan(
			&entity.ID,
			&entity.Description,
			&entity.Code,
			&entity.Name,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &result, nil
}
