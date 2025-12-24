package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

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
	EstimateCountDetailedTransactions(ctx context.Context, opts models.DetailedTransactionFilterOptions) (total int, err error)
	GetAllDetailedTransactionsBySummaryID(ctx context.Context, summaryID string, relatedSummaryID *string, refNumber string) ([]models.DetailedTransactionOut, error)
	GetLastFailedOrRejectedTransaction(ctx context.Context, transactionType string, paymentType string) (*models.FailedOrRejectedTransaction, error)
	HasPendingTransactionAfterFailedOrRejected(ctx context.Context, transactionType string, paymentType string, failedOrRejectedID string) (bool, error)
	HasInProgressTransaction(ctx context.Context, transactionType string, paymentType string) (bool, error)
	HasPendingTransactionBefore(ctx context.Context, transactionType string, paymentType string, transactionDate time.Time) (bool, error)
	UpdateActivationStatus(ctx context.Context, summaryID string, isActive bool) error
	GetSummaryDetailBySummaryIDAllStatus(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error)
	GetDetailedTransactionIDsWithMapping(ctx context.Context, opts models.DetailedTransactionFilterOptions) (map[string]string, []string, error)
	GetTransactionsByIDs(ctx context.Context, transactionIDs []string, refNumber string) ([]models.DetailedTransactionOut, error)
	GetDetailedTransactionsChunk(ctx context.Context, summaryID string, relatedSummaryID *string, refNumber string, lastID string, limit int) ([]models.DetailedTransactionCSVOut, error)
	GetAllDetailedTransactionsForDownloadOptimized(ctx context.Context, summaryID string, relatedSummaryID *string, refNumber string) ([]models.DetailedTransactionCSVOut, error)
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
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW())
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
		    mfs.transaction_type,
		    mfs.payment_type, 
		    mfs.created_at,
		    mfs.transaction_source_creation_date,
		    mfs.requested_date, 
		    mfs.actual_date,
			mfs.source_account,
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
		WHERE mfs.id = $1 AND mfs.is_active = TRUE
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

	queryHasPendingTransactionBefore = `
	SELECT EXISTS (
		SELECT 1
		FROM money_flow_summaries
		WHERE transaction_type = $1 
		  AND payment_type = $2
		  AND money_flow_status = 'PENDING'
		  AND transaction_source_creation_date < $3
		  AND is_active = TRUE
	)
`
	queryUpdateActivationStatus = `
		UPDATE money_flow_summaries 
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`

	queryGetSummaryDetailBySummaryIDAllStatus = `
		SELECT
		    mfs.id,
		    mfs.transaction_type,
		    mfs.payment_type, 
		    mfs.created_at,
		    mfs.transaction_source_creation_date,
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
		in.CreatedAt,
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

	// DEBUG LOG RESULT
	if len(result) > 0 {
		xlog.Info(ctx, "[PAGINATION-DEBUG-RESULT]",
			xlog.Int("count", len(result)),
			xlog.String("first_id", result[0].ID[:8]),
			xlog.String("first_created_at", result[0].CreatedAt.Format(time.RFC3339)),
			xlog.String("last_id", result[len(result)-1].ID[:8]),
			xlog.String("last_created_at", result[len(result)-1].CreatedAt.Format(time.RFC3339)))
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
		&result.TransactionType,
		&result.PaymentType,
		&result.CreatedDate,
		&result.TransactionSourceCreationDate,
		&result.RequestedDate,
		&result.ActualDate,
		&result.SourceAccountNumber,
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
		InnerJoin(`money_flow_summaries mfs ON mfs."id" = dmfs."summary_id"`).
		Where(sq.Eq{`mfs."is_active"`: true}).
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

// HasPendingTransactionBefore checks if there's any PENDING transaction before the given date
// with the same transaction_type and payment_type
func (mfr *moneyFlowRepository) HasPendingTransactionBefore(
	ctx context.Context,
	transactionType string,
	paymentType string,
	transactionDate time.Time,
) (bool, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var exists bool
	err = db.QueryRowContext(
		ctx,
		queryHasPendingTransactionBefore,
		transactionType,
		paymentType,
		transactionDate,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (mfr *moneyFlowRepository) UpdateActivationStatus(ctx context.Context, summaryID string, isActive bool) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryUpdateActivationStatus, summaryID, isActive)
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

func (mfr *moneyFlowRepository) GetSummaryDetailBySummaryIDAllStatus(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	err = db.QueryRowContext(ctx, queryGetSummaryDetailBySummaryIDAllStatus, summaryID).Scan(
		&result.ID,
		&result.TransactionType,
		&result.PaymentType,
		&result.CreatedDate,
		&result.TransactionSourceCreationDate,
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

// EstimateCountDetailedTransactions estimates total count using EXPLAIN query
// Much faster than actual COUNT for large datasets (milliseconds vs seconds)
func (mfr *moneyFlowRepository) EstimateCountDetailedTransactions(ctx context.Context, opts models.DetailedTransactionFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	queryBuilder := NewMoneyFlowQueryBuilder()
	explainSQL, args, err := queryBuilder.BuildEstimatedCountDetailedTransactionsQuery(opts)
	if err != nil {
		return 0, fmt.Errorf("failed to build explain query: %w", err)
	}

	var jsonResult string
	err = db.QueryRowContext(ctx, explainSQL, args...).Scan(&jsonResult)
	if err != nil {
		return 0, err
	}

	// Parse EXPLAIN JSON result
	estimated, err := parseExplainRows(jsonResult)
	if err != nil {
		xlog.Warn(ctx, "[ESTIMATE-COUNT] Failed to parse EXPLAIN result, falling back to 0",
			xlog.Err(err))
		return 0, nil
	}

	xlog.Info(ctx, "[ESTIMATE-COUNT] Got estimated count",
		xlog.Int("estimated_total", estimated),
		xlog.String("summary_id", opts.SummaryID))

	return estimated, nil
}

// parseExplainRows parses PostgreSQL EXPLAIN JSON output to extract row estimation
func parseExplainRows(jsonResult string) (int, error) {
	var explain []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonResult), &explain); err != nil {
		return 0, fmt.Errorf("failed to unmarshal EXPLAIN result: %w", err)
	}

	if len(explain) == 0 {
		return 0, fmt.Errorf("empty EXPLAIN result")
	}

	plan, ok := explain[0]["Plan"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid EXPLAIN structure: no Plan field")
	}

	// Get "Plan Rows" which is PostgreSQL's estimation
	planRows, ok := plan["Plan Rows"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid EXPLAIN structure: no Plan Rows field")
	}

	return int(planRows), nil
}

// GetDetailedTransactionIDsWithMapping - Get IDs with dmfs.id mapping
func (mfr *moneyFlowRepository) GetDetailedTransactionIDsWithMapping(ctx context.Context, opts models.DetailedTransactionFilterOptions) (map[string]string, []string, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	summaryIDs := []string{opts.SummaryID}
	if opts.RelatedFailedOrRejectedSummaryID != nil && *opts.RelatedFailedOrRejectedSummaryID != "" {
		summaryIDs = append(summaryIDs, *opts.RelatedFailedOrRejectedSummaryID)
	}

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(`dmfs."id"`, `dmfs."acuan_transaction_id"`).
		From("detailed_money_flow_summaries as dmfs").
		Where(sq.Eq{`dmfs."summary_id"`: summaryIDs})

	// Apply cursor pagination
	if opts.Cursor != nil {
		if opts.Cursor.IsBackward {
			query = query.Where(sq.Gt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" ASC`)
		} else {
			query = query.Where(sq.Lt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" DESC`)
		}
	} else {
		query = query.OrderBy(`dmfs."id" DESC`)
	}

	if opts.Limit > 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Map: transactionId -> dmfs.id (for cursor)
	idMapping := make(map[string]string)
	var transactionIDs []string

	for rows.Next() {
		var dmfsID, acuanTxID string
		if err := rows.Scan(&dmfsID, &acuanTxID); err != nil {
			return nil, nil, err
		}
		idMapping[acuanTxID] = dmfsID // mapping transaction_id -> dmfs.id
		transactionIDs = append(transactionIDs, acuanTxID)
	}

	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}

	return idMapping, transactionIDs, nil
}

// GetTransactionsByIDs
func (mfr *moneyFlowRepository) GetTransactionsByIDs(ctx context.Context, transactionIDs []string, refNumber string) ([]models.DetailedTransactionOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	if len(transactionIDs) == 0 {
		return []models.DetailedTransactionOut{}, nil
	}

	columns := []string{
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
		From("transaction t").
		Where(sq.Eq{`t."transactionId"`: transactionIDs})

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

	// Store in map first
	transactionMap := make(map[string]models.DetailedTransactionOut)

	for rows.Next() {
		var dt models.DetailedTransactionOut
		err = rows.Scan(
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
		transactionMap[dt.TransactionID] = dt
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// Return in original order (preserve dmfs.id order from first query)
	var result []models.DetailedTransactionOut
	for _, txID := range transactionIDs {
		if tx, exists := transactionMap[txID]; exists {
			result = append(result, tx)
		}
	}

	return result, nil
}

// GetDetailedTransactionsChunk - Get transactions in chunks for streaming download
func (mfr *moneyFlowRepository) GetDetailedTransactionsChunk(
	ctx context.Context,
	summaryID string,
	relatedSummaryID *string,
	refNumber string,
	lastID string,
	limit int,
) ([]models.DetailedTransactionCSVOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	// STEP 1: Get chunk of transaction IDs from small table
	summaryIDs := []string{summaryID}
	if relatedSummaryID != nil && *relatedSummaryID != "" {
		summaryIDs = append(summaryIDs, *relatedSummaryID)
	}

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(`dmfs."id"`, `dmfs."acuan_transaction_id"`).
		From("detailed_money_flow_summaries as dmfs").
		InnerJoin(`money_flow_summaries mfs ON mfs."id" = dmfs."summary_id"`).
		Where(sq.Eq{`mfs."is_active"`: true}).
		Where(sq.Eq{`dmfs."summary_id"`: summaryIDs}).
		OrderBy(`dmfs."id" ASC`). // Consistent ordering untuk cursor
		Limit(uint64(limit))

	// Cursor pagination untuk chunking
	if lastID != "" {
		query = query.Where(sq.Gt{`dmfs."id"`: lastID})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build chunk query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect IDs and mapping
	var transactionIDs []string
	idMapping := make(map[string]string) // transactionId -> dmfs.id

	for rows.Next() {
		var dmfsID, acuanTxID string
		if err := rows.Scan(&dmfsID, &acuanTxID); err != nil {
			return nil, err
		}
		transactionIDs = append(transactionIDs, acuanTxID)
		idMapping[acuanTxID] = dmfsID
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(transactionIDs) == 0 {
		return []models.DetailedTransactionCSVOut{}, nil
	}

	xlog.Info(ctx, "[DOWNLOAD-CHUNK] Retrieved chunk IDs",
		xlog.String("summary_id", summaryID),
		xlog.Int("chunk_size", len(transactionIDs)),
		xlog.String("last_id", lastID))

	// STEP 2: Fetch transactions for this chunk
	transactions, err := mfr.getTransactionsBatchForDownload(ctx, transactionIDs, refNumber, idMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions batch: %w", err)
	}

	return transactions, nil
}

// getTransactionsBatchForDownload fetches transactions and preserves order
func (mfr *moneyFlowRepository) getTransactionsBatchForDownload(
	ctx context.Context,
	transactionIDs []string,
	refNumber string,
	idMapping map[string]string,
) ([]models.DetailedTransactionCSVOut, error) {
	db := mfr.r.extractTxRead(ctx)

	columns := []string{
		`t."transactionId"`,
		`t."transactionDate"`,
		`t."refNumber"`,
		`t."typeTransaction"`,
		`t."fromAccount"`,
		`t."toAccount"`,
		`t."amount"`,
		`t."description"`,
		`COALESCE(t."metadata", '{}'::jsonb) as "metadata"`,
		`t."createdAt"`,
	}

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(columns...).
		From("transaction t").
		Where(sq.Eq{`t."transactionId"`: transactionIDs})

	if refNumber != "" {
		query = query.Where(sq.Eq{`t."refNumber"`: refNumber})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactionMap := make(map[string]models.DetailedTransactionCSVOut)

	for rows.Next() {
		var dt models.DetailedTransactionCSVOut
		err = rows.Scan(
			&dt.TransactionID,
			&dt.TransactionDate,
			&dt.RefNumber,
			&dt.TypeTransaction,
			&dt.SourceAccount,
			&dt.DestinationAccount,
			&dt.Amount,
			&dt.Description,
			&dt.Metadata,
			&dt.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if dmfsID, exists := idMapping[dt.TransactionID]; exists {
			dt.ID = dmfsID
		}

		transactionMap[dt.TransactionID] = dt
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// Return in original order
	var result []models.DetailedTransactionCSVOut
	for _, txID := range transactionIDs {
		if tx, exists := transactionMap[txID]; exists {
			result = append(result, tx)
		}
	}

	return result, nil
}

// GetAllDetailedTransactionsForDownloadOptimized - Optimized single query with safety limits
func (mfr *moneyFlowRepository) GetAllDetailedTransactionsForDownloadOptimized(
	ctx context.Context,
	summaryID string,
	relatedSummaryID *string,
	refNumber string,
) ([]models.DetailedTransactionCSVOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	// Build summary IDs
	summaryIDs := []string{summaryID}
	if relatedSummaryID != nil && *relatedSummaryID != "" {
		summaryIDs = append(summaryIDs, *relatedSummaryID)
	}

	// Optimized columns - only what we need
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
		`COALESCE(t."metadata", '{}') as "metadata"`,
		`t."createdAt"`,
	}

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(columns...).
		From("detailed_money_flow_summaries as dmfs").
		Join(`transaction t ON t."transactionId" = dmfs."acuan_transaction_id"`).
		Where(sq.Eq{`dmfs."summary_id"`: summaryIDs}).
		OrderBy(`dmfs."id" ASC`)

	if refNumber != "" {
		query = query.Where(sq.Eq{`t."refNumber"`: refNumber})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	xlog.Info(ctx, "[DOWNLOAD-QUERY] Executing query",
		xlog.String("summary_id", summaryID),
		xlog.String("ref_number", refNumber))

	startTime := time.Now()
	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Pre-allocate with reasonable estimate
	const estimatedRows = 100000
	const maxRows = 500000 // Safety limit
	result := make([]models.DetailedTransactionCSVOut, 0, estimatedRows)

	rowCount := 0
	lastLogTime := startTime

	for rows.Next() {
		// Safety check: max rows limit
		if rowCount >= maxRows {
			xlog.Error(ctx, "[DOWNLOAD-QUERY] Exceeded maximum row limit",
				xlog.String("summary_id", summaryID),
				xlog.Int("max_rows", maxRows))
			return nil, fmt.Errorf("data too large: exceeded maximum %d rows", maxRows)
		}

		var dt models.DetailedTransactionCSVOut
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
			&dt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row %d: %w", rowCount, err)
		}

		result = append(result, dt)
		rowCount++

		// Check context cancellation every 10K rows
		if rowCount%10000 == 0 {
			// Check for timeout
			elapsed := time.Since(startTime)
			if elapsed > 14*time.Second {
				xlog.Warn(ctx, "[DOWNLOAD-QUERY] Query taking too long",
					xlog.String("summary_id", summaryID),
					xlog.Int("rows_fetched", rowCount),
					xlog.Duration("elapsed", elapsed))
				return nil, fmt.Errorf("query timeout: fetched %d rows in %v (limit: 14s)", rowCount, elapsed)
			}

			// Check context
			select {
			case <-ctx.Done():
				xlog.Warn(ctx, "[DOWNLOAD-QUERY] Context cancelled",
					xlog.String("summary_id", summaryID),
					xlog.Int("rows_fetched", rowCount))
				return nil, fmt.Errorf("query cancelled: %w", ctx.Err())
			default:
				// Continue
			}

			// Log progress every 5 seconds
			if time.Since(lastLogTime) > 5*time.Second {
				xlog.Info(ctx, "[DOWNLOAD-QUERY] Progress",
					xlog.String("summary_id", summaryID),
					xlog.Int("rows_fetched", rowCount),
					xlog.Duration("elapsed", elapsed))
				lastLogTime = time.Now()
			}
		}
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row iteration error: %w", rows.Err())
	}

	queryDuration := time.Since(startTime)
	xlog.Info(ctx, "[DOWNLOAD-QUERY] Query completed",
		xlog.String("summary_id", summaryID),
		xlog.Int("total_rows", rowCount),
		xlog.Duration("query_duration", queryDuration),
		xlog.Float64("rows_per_second", float64(rowCount)/queryDuration.Seconds()))

	return result, nil
}
