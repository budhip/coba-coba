package repositories

import (
	"context"
	"errors"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type AccountConfigRepository interface {
	// GetWht2326 is get account number for wht2326. Used in RPYAC
	GetWht2326(ctx context.Context, loanAccountNumber string, loanType string) (string, error)

	// GetVatOut is get account number for vat out. Used in RPYAF
	GetVatOut(ctx context.Context, loanAccountNumber string, loanType string) (string, error)

	// GetRevenue is get account number for Amartha Revenue. Used in RPYAF
	GetRevenue(ctx context.Context, loanAccountNumber string, loanType string) (string, error)

	GetAdminFee(ctx context.Context, loanAccountNumber string, loanKind string) (string, error)
}

type accountConfigRepository sqlRepo

var _ AccountConfigRepository = (*accountConfigRepository)(nil)

func (a *accountConfigRepository) GetWht2326(ctx context.Context, loanAccountNumber string, loanType string) (an string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	account, err := a.r.GetAccountRepository().GetCachedAccount(
		ctx,
		loanAccountNumber,
	)
	if err != nil {
		return "", err
	}

	entity := a.r.config.AccountConfig.MapAccountEntity[account.Entity]
	if entity == "" {
		return "", common.ErrMissingEntityFromAccount
	}

	accountConfig := getMapFromConfig(a.r.config.AccountConfig.WHT2326ByEntityCode, entity)

	wht2326, err := getAccountNumberFromConfig(accountConfig, loanType)
	if err != nil {
		return "", err
	}

	return wht2326, err
}

func (a *accountConfigRepository) GetVatOut(ctx context.Context, loanAccountNumber string, loanType string) (an string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	account, err := a.r.GetAccountRepository().GetCachedAccount(
		ctx,
		loanAccountNumber,
	)
	if err != nil {
		return "", err
	}

	entity := a.r.config.AccountConfig.MapAccountEntity[account.Entity]
	if entity == "" {
		return "", common.ErrMissingEntityFromAccount
	}

	mapAccountVAT := getMapFromConfig(a.r.config.AccountConfig.VATOutByEntityCode, entity)

	VATOut, err := getAccountNumberFromConfig(mapAccountVAT, loanType)
	if err != nil {
		return "", err
	}

	return VATOut, nil
}

func (a *accountConfigRepository) GetRevenue(ctx context.Context, loanAccountNumber string, loanType string) (an string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	account, err := a.r.GetAccountRepository().GetCachedAccount(
		ctx,
		loanAccountNumber,
	)
	if err != nil {
		return "", err
	}

	entity := a.r.config.AccountConfig.MapAccountEntity[account.Entity]
	if entity == "" {
		return "", common.ErrMissingEntityFromAccount
	}

	mapAccountRevenue := getMapFromConfig(a.r.config.AccountConfig.AmarthaRevenueByEntityCode, entity)

	amarthaRevenue, err := getAccountNumberFromConfig(mapAccountRevenue, loanType)
	if err != nil {
		return "", err
	}

	return amarthaRevenue, nil
}

// add temporary for hot fix https://amartha.atlassian.net/browse/FN-128
// TODO replace with real implemtation
func (a *accountConfigRepository) GetAdminFee(ctx context.Context, loanAccountNumber string, loanKind string) (string, error) {
	// avoid silent error
	return "", errors.New("error: Getadminfee for local config is not available")
}
