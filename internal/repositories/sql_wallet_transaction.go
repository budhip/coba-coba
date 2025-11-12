package repositories

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type WalletTransactionRepository interface {
	Create(ctx context.Context, in models.NewWalletTransaction) (*models.WalletTransaction, error)
	GetById(ctx context.Context, id string) (*models.WalletTransaction, error)
	Update(ctx context.Context, id string, data models.WalletTransactionUpdate) (*models.WalletTransaction, error)
	GetByRefNumber(ctx context.Context, refNumber string) (*models.WalletTransaction, error)
	CheckTransactionTypeAndReferenceNumber(ctx context.Context, trxType, refNumber string) (*models.WalletTransaction, error)
	List(ctx context.Context, opts models.WalletTrxFilterOptions) ([]models.WalletTransaction, error)
	CountAll(ctx context.Context, opts models.WalletTrxFilterOptions) (total int, err error)
}

type walletTrxRepo sqlRepo

var _ WalletTransactionRepository = (*walletTrxRepo)(nil)

func (e *walletTrxRepo) Create(ctx context.Context, in models.NewWalletTransaction) (*models.WalletTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	args, err := common.GetFieldValues(in)
	if err != nil {
		return nil, err
	}

	var created models.WalletTransaction
	var destinationAccountNumber, description sql.NullString

	err = db.QueryRowContext(ctx, queryWalletTrxCreate, args...).
		Scan(
			&created.ID,
			&created.Status,
			&created.AccountNumber,
			&destinationAccountNumber,
			&created.RefNumber,
			&created.TransactionType,
			&created.TransactionTime,
			&created.TransactionFlow,
			&created.NetAmount,
			&created.Amounts,
			&description,
			&created.Metadata,
			&created.CreatedAt,
		)
	if err != nil {
		return nil, err
	}

	// TODO: change this if we already save the currency of NetAmount in the database
	created.NetAmount.Currency = models.IDRCurrency

	created.DestinationAccountNumber = destinationAccountNumber.String
	created.Description = description.String

	return &created, nil
}

func (e *walletTrxRepo) GetById(ctx context.Context, id string) (*models.WalletTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	var destinationAccountNumber, description sql.NullString
	var wt models.WalletTransaction

	err = db.QueryRowContext(ctx, queryWalletTrxGetByID, id).
		Scan(
			&wt.ID,
			&wt.Status,
			&wt.AccountNumber,
			&destinationAccountNumber,
			&wt.RefNumber,
			&wt.TransactionType,
			&wt.TransactionTime,
			&wt.TransactionFlow,
			&wt.NetAmount,
			&wt.Amounts,
			&description,
			&wt.Metadata,
			&wt.CreatedAt,
		)
	if err != nil {
		return nil, err
	}

	wt.DestinationAccountNumber = destinationAccountNumber.String
	wt.Description = description.String

	// TODO: change this if we already save the currency of NetAmount in the database
	wt.NetAmount.Currency = models.IDRCurrency

	return &wt, nil
}

func (e *walletTrxRepo) Update(ctx context.Context, id string, data models.WalletTransactionUpdate) (res *models.WalletTransaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	query, args, err := buildUpdateWalletTrx(id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var wt models.WalletTransaction
	var destinationAccountNumber, description sql.NullString

	err = db.QueryRowContext(ctx, query, args...).
		Scan(
			&wt.ID,
			&wt.Status,
			&wt.AccountNumber,
			&destinationAccountNumber,
			&wt.RefNumber,
			&wt.TransactionType,
			&wt.TransactionTime,
			&wt.TransactionFlow,
			&wt.NetAmount,
			&wt.Amounts,
			&description,
			&wt.Metadata,
			&wt.CreatedAt,
		)
	if err != nil {
		return nil, err
	}

	wt.DestinationAccountNumber = destinationAccountNumber.String
	wt.Description = description.String

	// TODO: change this if we already save the currency of NetAmount in the database
	wt.NetAmount.Currency = models.IDRCurrency

	return &wt, nil
}

func (e *walletTrxRepo) GetByRefNumber(ctx context.Context, refNumber string) (*models.WalletTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	var created models.WalletTransaction
	err = db.QueryRowContext(ctx, queryWalletTrxGetByRefNumber, refNumber).
		Scan(
			&created.ID,
			&created.Status,
		)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &created, nil
}

func (e *walletTrxRepo) List(ctx context.Context, opts models.WalletTrxFilterOptions) ([]models.WalletTransaction, error) {
	var err error
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	query, args, err := buildListWalletTrxQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var result []models.WalletTransaction
	for rows.Next() {
		var destinationAccountNumber, description sql.NullString
		var trx models.WalletTransaction

		var err = rows.Scan(
			&trx.ID,
			&trx.Status,
			&trx.AccountNumber,
			&destinationAccountNumber,
			&trx.RefNumber,
			&trx.TransactionType,
			&trx.TransactionTime,
			&trx.TransactionFlow,
			&trx.NetAmount,
			&trx.Amounts,
			&description,
			&trx.Metadata,
			&trx.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		trx.DestinationAccountNumber = destinationAccountNumber.String
		trx.Description = description.String

		result = append(result, trx)
	}
	if rows.Err() != nil {
		return result, err
	}

	return result, nil
}

func (e *walletTrxRepo) CountAll(ctx context.Context, opts models.WalletTrxFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	query, args, err := buildCountWalletTrxQuery(opts)
	if err != nil {
		return total, fmt.Errorf("failed to build query: %w", err)
	}
	if err = db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return
	}

	return
}

func (e *walletTrxRepo) CheckTransactionTypeAndReferenceNumber(ctx context.Context, trxType, refNumber string) (*models.WalletTransaction, error) {
	var err error
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := e.r.extractTxWrite(ctx)

	var data models.WalletTransaction
	var destinationAccountNumber, description sql.NullString
	err = db.QueryRowContext(ctx, queryWalletTrxGetByTransactionTypeAndRefNumber, trxType, refNumber).
		Scan(
			&data.ID,
			&data.AccountNumber,
			&data.RefNumber,
			&data.TransactionType,
			&data.TransactionFlow,
			&data.TransactionTime,
			&data.NetAmount,
			&data.Amounts,
			&data.Status,
			&destinationAccountNumber,
			&description,
			&data.Metadata,
			&data.CreatedAt,
		)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	data.DestinationAccountNumber = destinationAccountNumber.String
	data.Description = description.String

	return &data, nil
}
