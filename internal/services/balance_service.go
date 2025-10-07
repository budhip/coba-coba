package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type BalanceService interface {
	// Get balance of account based on accountNumber in PAS or t24 format
	Get(ctx context.Context, accountNumber string) (models.AccountBalance, error)
	AdjustAccountBalance(ctx context.Context, accountNumber string, updateAmount models.Decimal) error
}

type balance service

var _ BalanceService = (*balance)(nil)

func (b balance) Get(ctx context.Context, accountNumber string) (res models.AccountBalance, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	repoBalance := b.srv.sqlRepo.GetBalanceRepository()

	res, err = repoBalance.Get(ctx, accountNumber)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeyAccountNumberNotFound)
		return
	}

	return
}

func (b balance) AdjustAccountBalance(ctx context.Context, accountNumber string, delta models.Decimal) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	repoBalance := b.srv.sqlRepo.GetBalanceRepository()

	err = repoBalance.AdjustAccountBalance(ctx, accountNumber, delta)
	if err != nil {
		return
	}

	return nil
}
