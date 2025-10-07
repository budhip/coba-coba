package repositories

import (
	"context"
	"database/sql"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type SubCategoryRepository interface {
	CheckSubCategoryByCodeAndCategoryCode(ctx context.Context, code, categoryCode string) (err error)
	GetByCode(ctx context.Context, code string) (*models.SubCategory, error)
	Create(ctx context.Context, in *models.CreateSubCategory) (created *models.SubCategory, err error)
	GetAll(ctx context.Context) (*[]models.SubCategory, error)
}

type subCategoryRepository sqlRepo

var _ SubCategoryRepository = (*subCategoryRepository)(nil)

func (scr *subCategoryRepository) CheckSubCategoryByCodeAndCategoryCode(ctx context.Context, code, categoryCode string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := scr.r.extractTxWrite(ctx)

	var subCategoryCode string
	err = db.QueryRowContext(ctx, querySubCategoryIsExistByCode, code, categoryCode).Scan(
		&subCategoryCode,
	)
	if err != nil {
		return
	}

	return
}

// GetByCode retrieves a SubCategory by its code.
func (r *subCategoryRepository) GetByCode(ctx context.Context, code string) (*models.SubCategory, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	var subCategory models.SubCategory
	err = db.QueryRowContext(ctx, querySubCategoryGetByCode, code).Scan(
		&subCategory.ID,
		&subCategory.CategoryCode,
		&subCategory.Description,
		&subCategory.Code,
		&subCategory.Name,
		&subCategory.CreatedAt,
		&subCategory.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &subCategory, nil
}

// Create implements SubCategoryRepository.
func (r *subCategoryRepository) Create(ctx context.Context, in *models.CreateSubCategory) (created *models.SubCategory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	in.Name = strings.ToUpper(in.Name)
	args, err := common.GetFieldValues(*in)
	if err != nil {
		return
	}

	var subCat models.SubCategory
	err = db.QueryRowContext(ctx, querySubCategoryCreate, args...).Scan(
		&subCat.ID,
		&subCat.CategoryCode,
		&subCat.Code,
		&subCat.Name,
		&subCat.Description,
		&subCat.CreatedAt,
		&subCat.UpdatedAt,
	)
	if err != nil {
		return
	}
	created = &subCat

	return
}

// GetAll retrieves all SubCategory
func (r *subCategoryRepository) GetAll(ctx context.Context) (*[]models.SubCategory, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	var result []models.SubCategory
	rows, err := db.QueryContext(ctx, queryGetAllSubCategory)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var value models.SubCategory
		var err = rows.Scan(
			&value.ID,
			&value.CategoryCode,
			&value.Code,
			&value.Name,
			&value.Description,
			&value.CreatedAt,
			&value.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return &result, nil
}
