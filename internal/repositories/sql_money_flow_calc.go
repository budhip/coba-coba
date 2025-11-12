package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	sq "github.com/Masterminds/squirrel"

	"github.com/google/uuid"
)

type MoneyFlowRepository interface {
	CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (string, error)
	CreateDetailedSummary(ctx context.Context, in models.CreateDetailedMoneyFlowSummary) error
	GetTransactionProcessed(ctx context.Context, breakdownTransactionsFrom string, transactionSourceDate time.Time) (*models.MoneyFlowTransactionProcessed, error)
	UpdateSummary(ctx context.Context, summaryID string, update models.MoneyFlowSummaryUpdate) error
	GetSummaryIDByPapaTransactionID(ctx context.Context, papaTransactionID string) (string, error)
	GetSummariesList(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) ([]models.MoneyFlowSummaryOut, error)
	CountSummaryAll(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) (total int, err error)
	GetSummaryDetailBySummaryID(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error)
	GetDetailedTransactionsBySummaryID(ctx context.Context, opts models.DetailedTransactionFilterOptions) ([]models.DetailedTransactionOut, error)
	CountDetailedTransactions(ctx context.Context, opts models.DetailedTransactionFilterOptions) (total int, err error)
	GetAllDetailedTransactionsBySummaryID(ctx context.Context, summaryID string, relatedSummaryID *string, refNumber string) ([]models.DetailedTransactionOut, error)
	GetLastFailedOrRejectedTransaction(ctx context.Context, transactionType string, paymentType string) (*models.FailedOrRejectedTransaction, error)
	HasPendingTransactionAfterFailedOrRejected(ctx context.Context, transactionType string, paymentType string, failedOrRejectedID string) (bool, error)
	HasInProgressTransaction(ctx context.Context, transactionType string, paymentType string) (bool, error)
}

type moneyFlowRepository sqlRepo

var _ MoneyFlowRepository = (*moneyFlowRepository)(nil)

const (
	queryCreateSummary = `
	INSERT INTO money_flow_summaries (
		id, transaction_source_creation_date, transaction_type, payment_type, 
		reference_number, description, source_account, destination_account, 
		total_transfer, papa_transaction_id, money_flow_status, 
		requested_date, actual_date, 
		source_bank_account_number, source_bank_account_name, source_bank_name,
		destination_bank_account_number, destination_bank_account_name, destination_bank_name,
		related_failed_or_rejected_summary_id,
		created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())
	RETURNING id
`

	queryCreateDetailedSummary = `
		INSERT INTO detailed_money_flow_summaries (
			id, summary_id, acuan_transaction_id, created_at
		)
		VALUES ($1, $2, $3, NOW())
	`

	queryGetTransactionProcessed = `
		SELECT 
			id, transaction_source_creation_date, transaction_type,
			payment_type, total_transfer, money_flow_status
		FROM money_flow_summaries
		WHERE transaction_type = $1 AND transaction_source_creation_date = $2 AND money_flow_status = 'PENDING'
	`

	queryGetSummaryIDByPapaTransactionID = `
		SELECT id
		FROM money_flow_summaries
		WHERE papa_transaction_id = $1
		LIMIT 1
	`

	queryGetSummaryDetailBySummaryID = `
		SELECT
		    mfs.id, 
		    mfs.payment_type, 
		    mfs.created_at, 
		    mfs.requested_date, 
		    mfs.actual_date,
			mfs.total_transfer, 
			mfs.money_flow_status, 
			mfs.source_bank_account_number, 
			mfs.source_bank_account_name, 
			mfs.source_bank_name,
			mfs.destination_bank_account_number, 
			mfs.destination_bank_account_name, 
			mfs.destination_bank_name,
			mfs.related_failed_or_rejected_summary_id,
			COALESCE(related.total_transfer, 0) as related_total_transfer
		FROM money_flow_summaries mfs
		LEFT JOIN money_flow_summaries related ON mfs.related_failed_or_rejected_summary_id = related.id
		WHERE mfs.id = $1
	`

	queryGetLastFailedOrRejectedTransaction = `
		SELECT 
			id, transaction_source_creation_date, transaction_type,
			payment_type, total_transfer, money_flow_status, created_at
		FROM money_flow_summaries
		WHERE transaction_type = $1 
		  AND payment_type = $2
		  AND money_flow_status IN ('FAILED', 'REJECTED')
		ORDER BY transaction_source_creation_date DESC
		LIMIT 1
	`

	queryHasPendingTransactionAfterFailedOrRejected = `
		SELECT EXISTS (
			SELECT 1
			FROM money_flow_summaries
			WHERE transaction_type = $1 
			  AND payment_type = $2
			  AND money_flow_status = 'PENDING'
			  AND related_failed_or_rejected_summary_id = $3
		)
	`

	queryHasInProgressTransaction = `
	SELECT EXISTS (
		SELECT 1
		FROM money_flow_summaries
		WHERE transaction_type = $1 
		  AND payment_type = $2
		  AND money_flow_status = 'IN_PROGRESS'
	)
`
)

func (mfr *moneyFlowRepository) CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (string, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	err = db.QueryRowContext(ctx, queryCreateSummary,
		in.ID,
		in.TransactionSourceCreationDate,
		in.TransactionType,
		in.PaymentType,
		in.ReferenceNumber,
		in.Description,
		in.SourceAccount,
		in.DestinationAccount,
		in.TotalTransfer,
		in.PapaTransactionID,
		in.MoneyFlowStatus,
		in.RequestedDate,
		in.ActualDate,
		in.SourceBankAccountNumber,
		in.SourceBankAccountName,
		in.SourceBankName,
		in.DestinationBankAccountNumber,
		in.DestinationBankAccountName,
		in.DestinationBankName,
		in.RelatedFailedOrRejectedSummaryID,
	).Scan(&in.ID)

	if err != nil {
		return "", err
	}

	return in.ID, nil
}

func (mfr *moneyFlowRepository) CreateDetailedSummary(ctx context.Context, in models.CreateDetailedMoneyFlowSummary) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	id := uuid.New().String()
	res, err := db.ExecContext(ctx, queryCreateDetailedSummary,
		id,
		in.SummaryID,
		in.AcuanTransactionID,
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

func (mfr *moneyFlowRepository) GetTransactionProcessed(ctx context.Context, breakdownTransactionsFrom string, transactionSourceDate time.Time) (*models.MoneyFlowTransactionProcessed, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var result models.MoneyFlowTransactionProcessed
	err = db.QueryRowContext(ctx, queryGetTransactionProcessed, breakdownTransactionsFrom, transactionSourceDate).Scan(
		&result.ID,
		&result.TransactionSourceCreationDate,
		&result.TransactionType,
		&result.PaymentType,
		&result.TotalTransfer,
		&result.MoneyFlowStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}

func (mfr *moneyFlowRepository) UpdateSummary(ctx context.Context, summaryID string, updates models.MoneyFlowSummaryUpdate) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	// Build dynamic query
	setClauses := []string{}
	args := []interface{}{summaryID} // $1 for WHERE clause
	paramIndex := 2

	if updates.PaymentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("payment_type = $%d", paramIndex))
		args = append(args, *updates.PaymentType)
		paramIndex++
	}

	if updates.TotalTransfer != nil {
		setClauses = append(setClauses, fmt.Sprintf("total_transfer = $%d", paramIndex))
		args = append(args, *updates.TotalTransfer)
		paramIndex++
	}

	if updates.PapaTransactionID != nil {
		setClauses = append(setClauses, fmt.Sprintf("papa_transaction_id = $%d", paramIndex))
		args = append(args, *updates.PapaTransactionID)
		paramIndex++
	}

	if updates.MoneyFlowStatus != nil {
		setClauses = append(setClauses, fmt.Sprintf("money_flow_status = $%d", paramIndex))
		args = append(args, *updates.MoneyFlowStatus)
		paramIndex++
	}

	if updates.RequestedDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("requested_date = $%d", paramIndex))
		args = append(args, *updates.RequestedDate)
		paramIndex++
	}

	if updates.ActualDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("actual_date = $%d", paramIndex))
		args = append(args, *updates.ActualDate)
		paramIndex++
	}

	// If no fields are updated, return an error
	if len(setClauses) == 0 {
		return errors.New("Error No Field to Update")
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = NOW()")

	// Build final query
	query := fmt.Sprintf(
		"UPDATE money_flow_summaries SET %s WHERE id = $1",
		strings.Join(setClauses, ", "),
	)

	res, err := db.ExecContext(ctx, query, args...)
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

func (mfr *moneyFlowRepository) GetSummaryIDByPapaTransactionID(ctx context.Context, papaTransactionID string) (string, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var summaryID string
	err = db.QueryRowContext(ctx, queryGetSummaryIDByPapaTransactionID, papaTransactionID).Scan(&summaryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return summaryID, nil
}

func (mfr *moneyFlowRepository) GetSummariesList(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) ([]models.MoneyFlowSummaryOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	queryBuilder := NewMoneyFlowQueryBuilder()
	query, args, err := queryBuilder.BuildListQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.MoneyFlowSummaryOut
	for rows.Next() {
		var mfs models.MoneyFlowSummaryOut
		err = rows.Scan(
			&mfs.ID,
			&mfs.TransactionSourceCreationDate,
			&mfs.PaymentType,
			&mfs.TotalTransfer,
			&mfs.MoneyFlowStatus,
			&mfs.RequestedDate,
			&mfs.ActualDate,
			&mfs.CreatedAt,
			&mfs.RelatedFailedOrRejectedSummaryID,
			&mfs.RelatedTotalTransfer,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, mfs)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return result, nil
}

func (mfr *moneyFlowRepository) CountSummaryAll(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	queryBuilder := NewMoneyFlowQueryBuilder()
	query, args, err := queryBuilder.BuildCountQuery(opts)
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	err = db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (mfr *moneyFlowRepository) GetSummaryDetailBySummaryID(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	err = db.QueryRowContext(ctx, queryGetSummaryDetailBySummaryID, summaryID).Scan(
		&result.ID,
		&result.PaymentType,
		&result.CreatedDate,
		&result.RequestedDate,
		&result.ActualDate,
		&result.TotalAmount,
		&result.Status,
		&result.SourceBankAccountNumber,
		&result.SourceBankAccountName,
		&result.SourceBankName,
		&result.DestinationBankAccountNumber,
		&result.DestinationBankAccountName,
		&result.DestinationBankName,
		&result.RelatedFailedOrRejectedSummaryID,
		&result.RelatedTotalTransfer,
	)
	if err != nil {
		return
	}

	return
}

func (mfr *moneyFlowRepository) GetDetailedTransactionsBySummaryID(ctx context.Context, opts models.DetailedTransactionFilterOptions) ([]models.DetailedTransactionOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	queryBuilder := NewMoneyFlowQueryBuilder()
	query, args, err := queryBuilder.BuildDetailedTransactionsQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DetailedTransactionOut
	for rows.Next() {
		var dt models.DetailedTransactionOut
		err = rows.Scan(
			&dt.ID,
			&dt.TransactionID,
			&dt.TransactionDate,
			&dt.RefNumber,
			&dt.TypeTransaction,
			&dt.SourceAccount,
			&dt.DestinationAccount,
			&dt.Amount,
			&dt.Description,
			&dt.Metadata,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dt)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return result, nil
}

func (mfr *moneyFlowRepository) CountDetailedTransactions(ctx context.Context, opts models.DetailedTransactionFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	queryBuilder := NewMoneyFlowQueryBuilder()
	query, args, err := queryBuilder.BuildCountDetailedTransactionsQuery(opts)
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	err = db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

// GetAllDetailedTransactionsBySummaryID gets all detailed transactions without pagination
// Now supports fetching from both main summary and related failed/rejected summary
func (mfr *moneyFlowRepository) GetAllDetailedTransactionsBySummaryID(ctx context.Context, summaryID string, relatedSummaryID *string, refNumber string) ([]models.DetailedTransactionOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	columns := []string{
		`dmfs."id"`,
		`t."transactionId"`,
		`t."transactionDate"`,
		`t."refNumber"`,
		`t."typeTransaction"`,
		`t."fromAccount"`,
		`t."toAccount"`,
		`t."amount"`,
		`t."description"`,
		`COALESCE(t."metadata", '{}'::jsonb) as "metadata"`,
	}

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(columns...).
		From("detailed_money_flow_summaries as dmfs").
		InnerJoin(`transaction t ON t."transactionId" = dmfs.acuan_transaction_id`).
		OrderBy(`t."transactionDate" ASC, dmfs."id" ASC`)

	// Build WHERE condition to include both summaryID and relatedSummaryID if exists
	if relatedSummaryID != nil && *relatedSummaryID != "" {
		query = query.Where(sq.Or{
			sq.Eq{`dmfs."summary_id"`: summaryID},
			sq.Eq{`dmfs."summary_id"`: *relatedSummaryID},
		})
	} else {
		query = query.Where(sq.Eq{`dmfs."summary_id"`: summaryID})
	}

	// Add refNumber filter if provided
	if refNumber != "" {
		query = query.Where(sq.Eq{`t."refNumber"`: refNumber})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DetailedTransactionOut
	for rows.Next() {
		var dt models.DetailedTransactionOut
		err = rows.Scan(
			&dt.ID,
			&dt.TransactionID,
			&dt.TransactionDate,
			&dt.RefNumber,
			&dt.TypeTransaction,
			&dt.SourceAccount,
			&dt.DestinationAccount,
			&dt.Amount,
			&dt.Description,
			&dt.Metadata,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dt)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return result, nil
}

// GetLastFailedOrRejectedTransaction
func (mfr *moneyFlowRepository) GetLastFailedOrRejectedTransaction(
	ctx context.Context,
	transactionType string,
	paymentType string,
) (*models.FailedOrRejectedTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var result models.FailedOrRejectedTransaction
	err = db.QueryRowContext(
		ctx,
		queryGetLastFailedOrRejectedTransaction,
		transactionType,
		paymentType,
	).Scan(
		&result.ID,
		&result.TransactionSourceCreationDate,
		&result.TransactionType,
		&result.PaymentType,
		&result.TotalTransfer,
		&result.MoneyFlowStatus,
		&result.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}

func (mfr *moneyFlowRepository) HasPendingTransactionAfterFailedOrRejected(
	ctx context.Context,
	transactionType string,
	paymentType string,
	failedOrRejectedID string,
) (bool, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var exists bool
	err = db.QueryRowContext(
		ctx,
		queryHasPendingTransactionAfterFailedOrRejected,
		transactionType,
		paymentType,
		failedOrRejectedID,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (mfr *moneyFlowRepository) HasInProgressTransaction(ctx context.Context, transactionType string, paymentType string) (bool, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var exists bool
	err = db.QueryRowContext(
		ctx,
		queryHasInProgressTransaction,
		transactionType,
		paymentType,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
