package repositories

import (
	"context"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type MoneyFlowRepository interface {
	GetSummaryByTypeAndDate(ctx context.Context, transactionType string, transactionDate time.Time) (*models.MoneyFlowSummary, error)
	CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (uint64, error)
	UpdateSummary(ctx context.Context, in models.UpdateMoneyFlowSummary) error
	CreateDetailedSummary(ctx context.Context, in models.CreateDetailedMoneyFlowSummary) error
	IsTransactionProcessed(ctx context.Context, transactionID string) (bool, error)
}

type moneyFlowRepository sqlRepo

var _ MoneyFlowRepository = (*moneyFlowRepository)(nil)

const (
	queryGetSummaryByTypeAndDate = `
		SELECT id, transaction_type, transaction_date, total_transfer, created_at, updated_at
		FROM money_flow_summaries
		WHERE transaction_type = $1 AND transaction_date = $2
	`

	queryCreateSummary = `
		INSERT INTO money_flow_summaries (transaction_type, transaction_date, total_transfer, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`

	queryUpdateSummary = `
		UPDATE money_flow_summaries
		SET total_transfer = $1, updated_at = NOW()
		WHERE id = $2 AND transaction_type = $3 AND transaction_date = $4
	`

	queryCreateDetailedSummary = `
		INSERT INTO detailed_money_flow_summaries (summary_id, transaction_id, ref_number, amount, transaction_time, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`

	queryIsTransactionProcessed = `
		SELECT EXISTS(
			SELECT 1 FROM detailed_money_flow_summaries
			WHERE transaction_id = $1
		)
	`
)

func (mfr *moneyFlowRepository) GetSummaryByTypeAndDate(ctx context.Context, transactionType string, transactionDate time.Time) (*models.MoneyFlowSummary, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	var summary models.MoneyFlowSummary
	err = db.QueryRowContext(ctx, queryGetSummaryByTypeAndDate, transactionType, transactionDate).Scan(
		&summary.ID,
		&summary.TransactionType,
		&summary.TransactionDate,
		&summary.TotalTransfer,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (mfr *moneyFlowRepository) CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (uint64, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	var id uint64
	err = db.QueryRowContext(ctx, queryCreateSummary,
		in.TransactionType,
		in.TransactionDate,
		in.TotalTransfer,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (mfr *moneyFlowRepository) UpdateSummary(ctx context.Context, in models.UpdateMoneyFlowSummary) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryUpdateSummary,
		in.TotalTransfer,
		in.ID,
		in.TransactionType,
		in.TransactionDate,
	)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return common.ErrNoRowsAffected
	}

	return nil
}

func (mfr *moneyFlowRepository) CreateDetailedSummary(ctx context.Context, in models.CreateDetailedMoneyFlowSummary) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryCreateDetailedSummary,
		in.SummaryID,
		in.TransactionID,
		in.RefNumber,
		in.Amount,
		in.TransactionTime,
	)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return common.ErrNoRowsAffected
	}

	return nil
}

func (mfr *moneyFlowRepository) IsTransactionProcessed(ctx context.Context, transactionID string) (bool, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	var exists bool
	err = db.QueryRowContext(ctx, queryIsTransactionProcessed, transactionID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
