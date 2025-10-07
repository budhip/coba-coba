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
)

type AccountBalanceDailyRepository interface {
	ListByDate(ctx context.Context, date time.Time) (results *[]models.AccountBalanceDaily, err error)
	Create(ctx context.Context, in *[]models.AccountBalanceDaily) (err error)
	GetLast(ctx context.Context) (result *models.AccountBalanceDaily, err error)
}

type accountBalanceDailyRepository sqlRepo

var _ AccountBalanceDailyRepository = (*accountBalanceDailyRepository)(nil)

func (r *accountBalanceDailyRepository) ListByDate(ctx context.Context, date time.Time) (results *[]models.AccountBalanceDaily, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	rows, err := db.QueryContext(ctx, queryListByDate, date)
	if err != nil {
		return
	}

	tempResults := []models.AccountBalanceDaily{}
	results = &tempResults
	defer rows.Close()
	for rows.Next() {
		var abd models.AccountBalanceDaily
		err = rows.Scan(
			&abd.AccountNumber,
			&abd.Date,
			&abd.Balance,
		)
		if err != nil {
			return
		}
		tempResults = append(tempResults, abd)
	}
	if rows.Err() != nil {
		return
	}

	results = &tempResults
	err = nil

	return
}

func (r *accountBalanceDailyRepository) Create(ctx context.Context, in *[]models.AccountBalanceDaily) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	tx, err := r.r.dbWrite.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	const DEFAULT_BATCH_SIZE int = 500 // pg can handle max 65535
	chunkList := common.ChunkBy[models.AccountBalanceDaily](*in, DEFAULT_BATCH_SIZE)
	for _, chunk := range chunkList {
		valueStrings := []string{}
		valueArgs := []interface{}{}
		for _, req := range chunk {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, req.AccountNumber)
			valueArgs = append(valueArgs, req.Date)
			valueArgs = append(valueArgs, req.Balance)
		}

		queryCreateBulk := fmt.Sprintf(`
		INSERT INTO account_balance_daily(
			"accountNumber", "date", "balance"
		) VALUES %s ON CONFLICT ("accountNumber", "date")
		DO UPDATE SET "balance" = EXCLUDED."balance", "updatedAt" = now()`, strings.Join(valueStrings, ","))

		sqlStr := common.ReplaceSQL(queryCreateBulk, "?")
		if _, err = tx.ExecContext(ctx, sqlStr, valueArgs...); err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return
}

func (r *accountBalanceDailyRepository) GetLast(ctx context.Context) (result *models.AccountBalanceDaily, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := r.r.extractTxWrite(ctx)

	var abd models.AccountBalanceDaily
	err = db.QueryRowContext(ctx, queryABDGetLast).Scan(
		&abd.AccountNumber,
		&abd.Date,
		&abd.Balance,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = common.ErrDataNotFound
		}
		return
	}
	result = &abd

	return
}
