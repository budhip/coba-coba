package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

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
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW())
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
		SELECT 
			id
		FROM money_flow_summaries
		WHERE papa_transaction_id = $1
		LIMIT 1
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
	args := []interface{}{summaryID} // $1 untuk WHERE clause
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

	query, args, err := buildMoneyFlowSummaryQuery(opts)
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

	query, args, err := buildMoneyFlowSummaryCountQuery(opts)
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	err = db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func buildMoneyFlowSummaryQuery(opts models.MoneyFlowSummaryFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`mfs."id"`,
		`mfs."transaction_source_creation_date"`,
		`mfs."payment_type"`,
		`mfs."total_transfer"`,
		`mfs."money_flow_status"`,
		`mfs."requested_date"`,
		`mfs."actual_date"`,
		`mfs."created_at"`,
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select(columns...).From("money_flow_summaries as mfs")

	// Filter by transaction_source_creation_date < today
	query = query.Where(sq.Lt{`mfs."transaction_source_creation_date"`: time.Now().Truncate(24 * time.Hour)})

	if opts.PaymentType != "" {
		query = query.Where(sq.Eq{`mfs."payment_type"`: opts.PaymentType})
	}

	if opts.TransactionSourceCreationDate != nil {
		query = query.Where(sq.Eq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDate})
	}

	if opts.Status != "" {
		query = query.Where(sq.Eq{`mfs."money_flow_status"`: opts.Status})
	}

	if opts.Cursor != nil {
		if opts.Cursor.IsBackward {
			query = query.Where(sq.Lt{`mfs."id"`: opts.Cursor.ID})
			query = query.OrderBy(`mfs."id" ASC`)
		} else {
			query = query.Where(sq.Lt{`mfs."id"`: opts.Cursor.ID})
			query = query.OrderBy(`mfs."id" DESC`)
		}
	} else {
		query = query.OrderBy(`mfs."id" DESC`)
	}

	if opts.Limit > 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	return query.ToSql()
}

func buildMoneyFlowSummaryCountQuery(opts models.MoneyFlowSummaryFilterOptions) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select("COUNT(*)").From("money_flow_summaries as mfs")

	// Filter by transaction_source_creation_date < today
	query = query.Where(sq.Lt{`mfs."transaction_source_creation_date"`: time.Now().Truncate(24 * time.Hour)})

	if opts.PaymentType != "" {
		query = query.Where(sq.Eq{`mfs."payment_type"`: opts.PaymentType})
	}

	if opts.TransactionSourceCreationDate != nil {
		query = query.Where(sq.Eq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDate})
	}

	if opts.Status != "" {
		query = query.Where(sq.Eq{`mfs."money_flow_status"`: opts.Status})
	}

	return query.ToSql()
}
