package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

type AccountService interface {
	Create(ctx context.Context, in models.CreateAccount) (out models.CreateAccount, err error)
	GetList(ctx context.Context, opts models.AccountFilterOptions) (accounts []models.GetAccountOut, total int, err error)
	GetTotalBalance(ctx context.Context, opts models.AccountFilterOptions) (*decimal.Decimal, error)
	GetOneByAccountNumber(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error)
	GetACuanAccountNumber(ctx context.Context, accountNumber string) (updatedAccountNumber string, err error)
	GetOneByAccountNumberOrLegacyId(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error)
	Upsert(ctx context.Context, in models.AccountUpsert) (err error)
	Update(ctx context.Context, reqBody models.UpdateAccountIn) (result models.GetAccountOut, err error)
	UpdateBySubCategory(ctx context.Context, in models.UpdateAccountBySubCategoryIn) (err error)
	RemoveDuplicateAccountMigration(ctx context.Context, accountNumber string) (err error)
	Delete(ctx context.Context, accountNumber string) (err error)
}

type account service

var _ AccountService = (*account)(nil)

func (as *account) Create(ctx context.Context, in models.CreateAccount) (out models.CreateAccount, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	in.IsHVT = slices.Contains(as.srv.conf.AccountConfig.HVTSubCategoryCodes, in.SubCategoryCode)

	if err = as.srv.sqlRepo.GetAccountRepository().Create(ctx, in); err != nil {
		err = checkDatabaseError(err)
		return
	}

	out = models.CreateAccount(in)

	return
}

func (as *account) GetList(ctx context.Context, opts models.AccountFilterOptions) (accounts []models.GetAccountOut, total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()

	accounts, err = accRepo.GetList(ctx, opts)
	if err != nil {
		return accounts, total, err
	}

	if len(accounts) == 0 {
		return accounts, total, nil
	}

	total, err = accRepo.CountAll(ctx, opts)
	if err != nil {
		return
	}

	return accounts, total, nil
}

// GetOneByAccountNumber will search account by it's account number then parse it to AccountResponse.
func (as *account) GetOneByAccountNumber(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	result, err = as.srv.sqlRepo.GetAccountRepository().GetOneByAccountNumber(ctx, accountNumber)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeyAccountNumberNotFound)
		return
	}

	return result, nil
}

func (as *account) Upsert(ctx context.Context, in models.AccountUpsert) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	if in.Status == "" {
		in.Status = common.MapAccountStatus[common.ACCOUNT_STATUS_ACTIVE]
	}

	in.IsHVT = slices.Contains(as.srv.conf.AccountConfig.HVTSubCategoryCodes, in.SubCategoryCode)

	if err = as.srv.sqlRepo.GetAccountRepository().Upsert(ctx, in); err != nil {
		return
	}

	return nil
}

// GetTotalBalance implements AccountService.
func (as *account) GetTotalBalance(ctx context.Context, opts models.AccountFilterOptions) (*decimal.Decimal, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()
	totalBalance, err := accRepo.GetTotalBalance(ctx, opts)
	if err != nil {
		return nil, err
	}

	return totalBalance, nil
}

func (as *account) GetACuanAccountNumber(ctx context.Context, accountNumber string) (updatedAccountNumber string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()

	// Check account in legacy id
	existInLegacyId, err := accRepo.GetOneByLegacyId(ctx, accountNumber)
	if err != nil && !errors.Is(sql.ErrNoRows, err) {
		return accountNumber, err
	}
	if existInLegacyId != nil {
		return existInLegacyId.AccountNumber, nil
	}

	// Check account in account number
	existInAccountNumber, err := accRepo.GetOneByAccountNumber(ctx, accountNumber)
	if err != nil {
		return accountNumber, err
	}
	return existInAccountNumber.AccountNumber, nil
}

// GetOneByAccountNumber will search account by it's account number then parse it to AccountResponse.
func (as *account) GetOneByAccountNumberOrLegacyId(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	result, err = as.srv.sqlRepo.GetAccountRepository().GetOneByAccountNumberOrLegacyId(ctx, accountNumber)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeyAccountNumberNotFound)
		return
	}

	return result, nil
}

// Update will search account by accountNumber then update the data.
func (as *account) Update(ctx context.Context, in models.UpdateAccountIn) (current models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	err = as.srv.sqlRepo.Atomic(ctx, func(ctx context.Context, r repositories.SQLRepository) error {
		accRepo := r.GetAccountRepository()
		featRepo := r.GetFeatureRepository()
		var errAtomic error

		// Check
		current, errAtomic = accRepo.GetOneByAccountNumber(ctx, in.AccountNumber)
		if errAtomic != nil {
			errAtomic = checkDatabaseError(errAtomic, models.ErrKeyAccountNumberNotFound)
			return errAtomic
		}

		errAtomic = accRepo.Update(ctx, current.ID, in)
		if errAtomic != nil {
			errAtomic = fmt.Errorf("unable to update account: %w", errAtomic)
			return errAtomic
		}

		// Update Features
		walletFeature := models.WalletOut{}
		walletFeature, errAtomic = featRepo.Update(ctx, &models.UpdateWalletIn{
			AccountNumber: in.AccountNumber,
			Feature:       in.Feature,
		})
		if errAtomic != nil {
			errAtomic = fmt.Errorf("unable to update feature: %w", errAtomic)
			return errAtomic
		}
		current.Features = walletFeature.Feature

		current.IsHVT = in.IsHVT != nil && *in.IsHVT
		if in.Status != "" {
			current.Status = in.Status
		}
		return nil
	})
	return
}

// RemoveDuplicateAccountMigration will first check if accountNumber is registered twice.
// if accountNumber is registered, get the second account by 1st account's legacyID.
// And if legacyID account found, delete it.
func (as *account) RemoveDuplicateAccountMigration(ctx context.Context, accountNumber string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()

	// Check account by accountNumber
	existByAccountNumber, err := accRepo.GetOneByAccountNumber(ctx, accountNumber)
	if err != nil {
		// account not registered, return
		if errors.Is(sql.ErrNoRows, err) {
			return nil
		}
		return fmt.Errorf("unable to GetOneByAccountNumber accountNumber %s: %w", accountNumber, err)
	}

	// account not registered, return
	if existByAccountNumber.ID <= 0 {
		return nil
	}

	// legacyId not found, return
	if existByAccountNumber.LegacyId == nil {
		return nil
	}

	legacyIDMap := *existByAccountNumber.LegacyId

	// t24AccountNumber not exist, return
	t24AccountNumber, ok := legacyIDMap["t24AccountNumber"].(string)
	if !ok {
		return nil
	}

	// t24AccountNumber empty, return
	if t24AccountNumber == "" || t24AccountNumber == "0" {
		return nil
	}

	// Account is registered, continue check 2nd account

	// Check account in account number
	existByLegacyID, err := accRepo.GetOneByAccountNumber(ctx, t24AccountNumber)
	if err != nil {
		// account not registered, return
		if errors.Is(sql.ErrNoRows, err) {
			return nil
		}
		return fmt.Errorf("unable to GetOneByAccountNumber legacyID %s: %w", t24AccountNumber, err)
	}

	// account not registered, return
	if existByLegacyID.ID <= 0 {
		return nil
	}

	// same account found, return
	if existByAccountNumber.ID == existByLegacyID.ID {
		return nil
	}

	// delete account that registered by legacyID
	if err = accRepo.Delete(ctx, existByLegacyID.ID); err != nil {
		return fmt.Errorf("unable to delete account by %d: %w", existByLegacyID.ID, err)
	}

	return nil
}

func (as *account) UpdateBySubCategory(ctx context.Context, in models.UpdateAccountBySubCategoryIn) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()

	err = accRepo.UpdateBySubCategory(ctx, in)
	if err != nil {
		return fmt.Errorf("unable to update account with sub category %s: %w", in.Code, err)
	}

	return nil
}

func (as *account) Delete(ctx context.Context, accountNumber string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accRepo := as.srv.sqlRepo.GetAccountRepository()

	return accRepo.DeleteByAccountNumber(ctx, accountNumber)
}
