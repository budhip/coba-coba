package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type CategoryRepository interface {
	CheckCategoryByCode(ctx context.Context, code string) (err error)
	GetCategorySequenceCode(ctx context.Context, code string) (seq int64, err error)
	Create(ctx context.Context, in *models.CreateCategoryIn) (created *models.Category, err error)
	GetByCode(ctx context.Context, code string) (*models.Category, error)
	List(ctx context.Context) (*[]models.Category, error)
}

type categoryRepository sqlRepo

var _ CategoryRepository = (*categoryRepository)(nil)

func (cr *categoryRepository) CheckCategoryByCode(ctx context.Context, code string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := cr.r.extractTxWrite(ctx)

	var categoryCode string
	err = db.QueryRowContext(ctx, queryCategoryIsExistByCode, code).Scan(
		&categoryCode,
	)
	if err != nil {
		return
	}

	return
}

func (cr *categoryRepository) GetCategorySequenceCode(ctx context.Context, code string) (seq int64, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := cr.r.extractTxWrite(ctx)
	sequenceName := fmt.Sprintf("category_code_%s_seq", code)

	err = db.QueryRowContext(ctx, queryCategoryGetSequence, sequenceName).Scan(
		&seq,
	)
	if err != nil {
		return
	}

	return
}

// Create implements CategoryRepository.
func (r *categoryRepository) Create(ctx context.Context, in *models.CreateCategoryIn) (*models.Category, error) {
	var err error
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	in.Name = strings.ToUpper(in.Name)
	args, err := common.GetFieldValues(*in)
	if err != nil {
		return nil, err
	}

	var result models.Category
	err = db.QueryRowContext(ctx, queryCategoryCreate, args...).Scan(
		&result.ID,
		&result.Code,
		&result.Name,
		&result.Description,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetByCode implements CategoryRepository.
func (r *categoryRepository) GetByCode(ctx context.Context, code string) (*models.Category, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)
	var category models.Category
	err = db.QueryRowContext(ctx, queryCategoryGetByCode, code).Scan(
		&category.ID,
		&category.Description,
		&category.Code,
		&category.Name,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

// List implements CategoryRepository.
func (r *categoryRepository) List(ctx context.Context) (*[]models.Category, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	// Execute the query with QueryContext
	rows, err := db.QueryContext(ctx, queryCategoryList)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the result set and process the data
	var result []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(
			&category.ID,
			&category.Description,
			&category.Code,
			&category.Name,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, category)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &result, nil
}
