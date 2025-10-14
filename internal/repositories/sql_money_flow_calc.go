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

	"github.com/google/uuid"
)

type MoneyFlowRepository interface {
	CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (string, error)
	CreateDetailedSummary(ctx context.Context, in models.CreateDetailedMoneyFlowSummary) error
	GetTransactionProcessed(ctx context.Context, breakdownTransactionsFrom string, transactionSourceDate time.Time) (*models.MoneyFlowTransactionProcessed, error)
	GetBankConfig(ctx context.Context, breakdownTransactionType string) (*models.BankConfig, error)
	UpdateSummary(ctx context.Context, summaryID string, update models.MoneyFlowSummaryUpdate) error
}

type moneyFlowRepository sqlRepo

var _ MoneyFlowRepository = (*moneyFlowRepository)(nil)

const (
	queryCreateSummary = `
		INSERT INTO money_flow_summaries (
			id, transaction_source_date, transaction_type, payment_type, 
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
			id, transaction_source_date, transaction_type,
			payment_type, total_transfer, money_flow_status
		FROM money_flow_summaries
		WHERE transaction_type = $1 AND transaction_source_date = $2
	`

	queryGetBankConfig = `
		SELECT 
			payment_type, transaction_type, breakdown_transaction_from,
			source_account_number, source_bank_name, 
			source_bank_account_number, source_bank_account_name,
			destination_account_number, 
			destination_bank_account_number, destination_bank_account_name, destination_bank_name
		FROM money_flow_bank_config
		WHERE breakdown_transaction_from = $1
	`
)

func (mfr *moneyFlowRepository) CreateSummary(ctx context.Context, in models.CreateMoneyFlowSummary) (string, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxWrite(ctx)

	id := uuid.New().String()
	var returnedID string
	err = db.QueryRowContext(ctx, queryCreateSummary,
		id,
		in.TransactionSourceDate,
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
	).Scan(&returnedID)

	if err != nil {
		return "", err
	}

	return returnedID, nil
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
		&result.TransactionSourceDate,
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

func (mfr *moneyFlowRepository) GetBankConfig(ctx context.Context, breakdownTransactionType string) (*models.BankConfig, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := mfr.r.extractTxRead(ctx)

	var config models.BankConfig
	err = db.QueryRowContext(ctx, queryGetBankConfig, breakdownTransactionType).Scan(
		&config.PaymentType,
		&config.TransactionType,
		&config.BreakdownTransactionFrom,
		&config.SourceAccountNumber,
		&config.SourceBankName,
		&config.SourceBankAccountNumber,
		&config.SourceBankAccountName,
		&config.DestinationAccountNumber,
		&config.DestinationBankAccountNumber,
		&config.DestinationBankAccountName,
		&config.DestinationBankName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &config, nil
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
