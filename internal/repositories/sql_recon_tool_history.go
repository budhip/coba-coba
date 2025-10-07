package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	xlog "bitbucket.org/Amartha/go-x/log"
)

type ReconToolHistoryRepository interface {
	Create(ctx context.Context, in *models.CreateReconToolHistoryIn) (created *models.ReconToolHistory, err error)
	GetList(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (result []models.ReconToolHistory, err error)
	DeleteByID(ctx context.Context, id string) error
	GetById(ctx context.Context, id uint64) (result *models.ReconToolHistory, err error)
	Update(ctx context.Context, id uint64, in *models.ReconToolHistory) (updated *models.ReconToolHistory, err error)
	CountAll(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (total int, err error)
}

type reconToolHistoryRepo sqlRepo

var _ ReconToolHistoryRepository = (*reconToolHistoryRepo)(nil)

func (r *reconToolHistoryRepo) Create(ctx context.Context, in *models.CreateReconToolHistoryIn) (created *models.ReconToolHistory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	args, err := common.GetFieldValues(*in)
	if err != nil {
		return
	}

	var entity models.ReconToolHistory
	err = db.QueryRowContext(ctx, queryReconToolHistoryCreate, args...).Scan(
		&entity.ID,
		&entity.TransactionDate,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		return
	}

	entity.OrderType = in.OrderType
	entity.TransactionType = in.TransactionType
	entity.UploadedFilePath = in.UploadedFilePath
	entity.Status = in.Status
	created = &entity

	return
}

func (r *reconToolHistoryRepo) GetList(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (result []models.ReconToolHistory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	query, args, err := buildListReconToolHistoryQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		var rth models.ReconToolHistory
		err = rows.Scan(
			&rth.ID,
			&rth.OrderType,
			&rth.TransactionType,
			&rth.TransactionDate,
			&rth.ResultFilePath,
			&rth.UploadedFilePath,
			&rth.Status,
			&rth.CreatedAt,
			&rth.UpdatedAt,
		)
		if err != nil {
			return result, err
		}
		result = append(result, rth)
	}
	if rows.Err() != nil {
		return result, err
	}

	return result, nil
}

func (r *reconToolHistoryRepo) CountAll(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	query, args, err := buildCountReconToolHistoryQuery(opts)
	if err != nil {
		return total, fmt.Errorf("failed to build query: %w", err)
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return
	}

	return
}

func (r *reconToolHistoryRepo) GetById(ctx context.Context, id uint64) (result *models.ReconToolHistory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	result = &models.ReconToolHistory{}
	err = db.QueryRowContext(ctx, queryReconToolHistoryGetById, id).Scan(
		&result.ID,
		&result.OrderType,
		&result.TransactionType,
		&result.TransactionDate,
		&result.ResultFilePath,
		&result.UploadedFilePath,
		&result.Status,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, common.ErrDataNotFound
		}
		return nil, err
	}

	return result, nil
}

func (r *reconToolHistoryRepo) DeleteByID(ctx context.Context, id string) error {
	var err error
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	result, err := db.ExecContext(ctx, queryReconToolHistoryDeleteByID, id)
	if err != nil {
		return err
	}

	// Check the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected <= 0 {
		xlog.Warnf(ctx, "no row affected on delete id: %s", id)
	}

	return nil
}

func (r *reconToolHistoryRepo) Update(ctx context.Context, id uint64, in *models.ReconToolHistory) (updated *models.ReconToolHistory, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	args := []any{
		id,
		in.OrderType,
		in.TransactionType,
		in.TransactionDate,
		in.UploadedFilePath,
		in.ResultFilePath,
		in.Status,
	}
	result, err := db.ExecContext(ctx, queryReconToolHistoryUpdate, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		err = common.ErrNoRowsAffected
		return
	}

	return in, nil
}
