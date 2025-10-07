package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/cache"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type AccountRepository interface {
	Create(ctx context.Context, in models.CreateAccount) (err error)
	GetList(ctx context.Context, opts models.AccountFilterOptions) (result []models.GetAccountOut, err error)
	GetAllWithoutPagination(ctx context.Context) (result *[]models.Account, err error)
	GetAllByAccountNumbers(ctx context.Context, accountNumbers []string) (result []models.Account, err error)
	CountAll(ctx context.Context, opts models.AccountFilterOptions) (total int, err error)
	CheckAccountNumbers(ctx context.Context, accountNumbers []string) (exists map[string]bool, err error)
	CheckDataByID(ctx context.Context, id uint64) (err error)
	Upsert(ctx context.Context, en models.AccountUpsert) (err error)
	Update(ctx context.Context, id int, newData models.UpdateAccountIn) (err error)
	UpdateBySubCategory(ctx context.Context, in models.UpdateAccountBySubCategoryIn) (err error)
	GetOneByAccountNumber(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error)
	GetCachedAccount(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error)
	GetOneByLegacyId(ctx context.Context, legacyId string) (*models.Account, error)
	Delete(ctx context.Context, accountID int) (err error)
	DeleteByAccountNumber(ctx context.Context, accountNumber string) (err error)
	GetOneByAccountNumberOrLegacyId(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error)

	// GetAccountBalances returns a map of account number to balance.
	// Deprecated: use BalanceRepository.GetMany instead.
	GetAccountBalances(ctx context.Context, req models.GetAccountBalanceRequest) (map[string]models.Balance, error)

	// GetTotalBalance returns the total balance of all accounts that match the filter options.
	// Deprecated: please rewrite this into the BalanceRepository.
	GetTotalBalance(ctx context.Context, opts models.AccountFilterOptions) (*decimal.Decimal, error)

	// UpdateAccountBalance updates the balance of an account.
	// Deprecated: please rewrite this into the BalanceRepository.
	UpdateAccountBalance(ctx context.Context, accountNumber string, balance models.Balance) (res *models.Balance, err error)
}

type accountRepository sqlRepo

var _ AccountRepository = (*accountRepository)(nil)

func (ar *accountRepository) Create(ctx context.Context, in models.CreateAccount) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	args, err := common.GetFieldValues(in)
	if err != nil {
		return
	}

	res, err := db.ExecContext(ctx, queryAccountCreate, args...)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		err = common.ErrNoRowsAffected
		return
	}

	return
}

func (ar *accountRepository) GetList(ctx context.Context, opts models.AccountFilterOptions) (result []models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	query, args, err := buildListAccountQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		var account models.GetAccountOut

		var actualBalance, pendingBalance decimal.Decimal
		err = rows.Scan(
			&account.ID,
			&account.AccountNumber,
			&account.OwnerID,
			&account.Category,
			&account.SubCategory,
			&account.Entity,
			&account.Currency,
			&actualBalance,
			&pendingBalance,
			&account.Status,
			&account.CreatedAt,
			&account.UpdatedAt,
			&account.AccountName,
		)
		if err != nil {
			return result, err
		}

		account.Balance = models.NewBalance(actualBalance, pendingBalance, models.WithBalanceLimitEnabled(ar.r.flag.IsEnabled(ar.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)))

		result = append(result, account)
	}
	if rows.Err() != nil {
		return result, err
	}

	return result, nil
}

func (ar *accountRepository) GetAllWithoutPagination(ctx context.Context) (results *[]models.Account, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	rows, err := db.QueryContext(ctx, queryAccountGetAllWithoutPagination)
	if err != nil {
		return
	}

	tempResults := []models.Account{}
	results = &tempResults
	defer rows.Close()
	for rows.Next() {
		var account models.Account
		err = rows.Scan(
			&account.AccountNumber,
			&account.OwnerID,
			&account.ActualBalance,
			&account.PendingBalance,
		)
		if err != nil {
			return
		}
		tempResults = append(tempResults, account)
	}
	if rows.Err() != nil {
		return
	}

	results = &tempResults
	err = nil

	return
}

func (ar *accountRepository) GetAllByAccountNumbers(ctx context.Context, accountNumbers []string) (results []models.Account, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	rows, err := db.QueryContext(ctx, queryGetAccountBalance, pq.Array(accountNumbers))
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		var account models.Account
		err = rows.Scan(
			&account.AccountNumber,
			&account.ActualBalance,
			&account.PendingBalance,
			&account.Name,
			&account.ProductTypeName,
			&account.SubCategoryCode,
		)
		if err != nil {
			return
		}

		results = append(results, account)
	}
	if rows.Err() != nil {
		return
	}

	return
}

// GetOneByAccountNumber will search account by it's account number on database.
func (ar *accountRepository) GetOneByAccountNumber(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	var (
		actualBalance, pendingBalance                          decimal.Decimal
		balanceRangeMin, negativeBalanceLimit, balanceRangeMax decimal.NullDecimal
		negativeBalanceAllowed                                 sql.NullBool
		featurePreset                                          sql.NullString
	)

	result.Features = &models.WalletFeature{}

	err = db.QueryRowContext(ctx, GetOneByAccountNumber, accountNumber).Scan(
		&result.ID,
		&result.AccountNumber,
		&result.OwnerID,
		&result.Category,
		&result.SubCategory,
		&result.Entity,
		&result.Currency,
		&result.Status,
		&result.IsHVT,
		&actualBalance,
		&pendingBalance,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.LegacyId,
		&featurePreset,
		&balanceRangeMin,
		&balanceRangeMax,
		&negativeBalanceAllowed,
		&negativeBalanceLimit,
		&result.AccountName,
	)
	if err != nil {
		return
	}

	result.Balance = models.NewBalance(actualBalance, pendingBalance, models.WithBalanceLimitEnabled(ar.r.flag.IsEnabled(ar.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)))
	result.Features.BalanceRangeMin = &balanceRangeMin.Decimal
	result.Features.BalanceRangeMax = &balanceRangeMax.Decimal
	result.Features.AllowedNegativeBalance = &negativeBalanceAllowed.Bool
	result.Features.NegativeBalanceLimit = &negativeBalanceLimit.Decimal
	featurePreset.String = strings.ToUpper(featurePreset.String)
	result.Features.Preset = &featurePreset.String

	return
}

func (ar *accountRepository) GetOneByLegacyId(ctx context.Context, legacyId string) (*models.Account, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	acc := models.Account{}
	err = db.QueryRowContext(ctx, queryGetOneByLegacyId, legacyId).Scan(
		&acc.ID,
		&acc.AccountNumber,
		&acc.ActualBalance,
		&acc.PendingBalance,
	)
	if err != nil {
		return nil, err
	}

	return &acc, nil
}

func (ar *accountRepository) CountAll(ctx context.Context, opts models.AccountFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	// TODO: change this query using estimation count by using explain analyze
	// 		 or change product requirement to show on FE as "more than 1000 data"
	//query, args, err := buildCountAccountQuery(opts)
	//if err != nil {
	//	return total, fmt.Errorf("failed to build query: %w", err)
	//}

	if err = db.QueryRowContext(ctx, queryEstimateCountAccount).Scan(&total); err != nil {
		return
	}

	return
}

func (ar *accountRepository) CheckDataByID(ctx context.Context, id uint64) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	if err = db.QueryRowContext(ctx, QueryAccountCheckDataById, id).Scan(&id); err != nil {
		return
	}

	return
}

func (ar *accountRepository) Upsert(ctx context.Context, en models.AccountUpsert) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	args := []any{
		en.AccountNumber,
		en.Name,
		en.OwnerID,
		en.ProductTypeName,
		en.CategoryCode,
		en.SubCategoryCode,
		en.EntityCode,
		en.Currency,
		en.AltID,
		en.LegacyId,
		en.IsHVT,
		en.Status,
		en.Metadata,
	}

	res, err := db.ExecContext(ctx, queryAccountUpsert, args...)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		err = common.ErrNoRowsAffected
		return
	}
	return
}

func (ar *accountRepository) CheckAccountNumbers(ctx context.Context, accountNumbers []string) (exists map[string]bool, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	exists = make(map[string]bool)
	for _, an := range accountNumbers {
		exists[an] = false
	}

	db := ar.r.extractTxWrite(ctx)

	rows, err := db.QueryContext(ctx, queryCheckByAccountNumbers, pq.Array(accountNumbers))
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var an string
		err = rows.Scan(&an)
		if err != nil {
			return nil, err
		}

		exists[an] = true
	}

	return exists, nil
}

func (ar *accountRepository) GetAccountBalances(ctx context.Context, req models.GetAccountBalanceRequest) (map[string]models.Balance, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accountBalance := make(map[string]models.Balance)

	db := ar.r.extractTxWrite(ctx)

	sql, args, err := buildGetAccountBalancesQuery(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		out := struct {
			AccountNumber  string
			ActualBalance  decimal.Decimal
			PendingBalance decimal.Decimal
			Version        int
			UpdatedAt      time.Time
		}{}

		err = rows.Scan(
			&out.AccountNumber,
			&out.ActualBalance,
			&out.PendingBalance,
			&out.Version,
			&out.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		accountBalance[out.AccountNumber] = models.NewBalance(
			out.ActualBalance,
			out.PendingBalance,
			models.WithVersion(out.Version),
			models.WithLastUpdatedAt(out.UpdatedAt),
			models.WithBalanceLimitEnabled(ar.r.flag.IsEnabled(ar.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)),
		)
	}

	return accountBalance, nil
}

func (ar *accountRepository) UpdateAccountBalance(ctx context.Context, accountNumber string, balance models.Balance) (updatedBalance *models.Balance, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	var version int
	if err = db.QueryRowContext(ctx, queryGetAccountVersion, accountNumber).Scan(
		&version,
	); err != nil {
		return
	}

	updatedAt := time.Now()

	res, err := db.ExecContext(ctx, queryUpdateAccountBalance,
		balance.Actual(),
		balance.Pending(),
		version+1,
		updatedAt,
		accountNumber,
		version,
	)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		return nil, common.ErrNoRowsAffected
	}

	ub := models.NewBalance(
		balance.Actual(),
		balance.Pending(),
		models.WithVersion(version+1),
		models.WithLastUpdatedAt(updatedAt),
		models.WithBalanceLimitEnabled(ar.r.flag.IsEnabled(ar.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)),
	)

	return &ub, nil
}

// GetTotalBalance implements AccountRepository.
func (ar *accountRepository) GetTotalBalance(ctx context.Context, opts models.AccountFilterOptions) (*decimal.Decimal, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	query, args, err := buildTotalBalanceAccountQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	totalBalance := decimal.NewFromFloat(0)
	err = db.QueryRowContext(ctx, query, args...).Scan(&totalBalance)
	if err != nil {
		return nil, err
	}

	return &totalBalance, nil
}

// Update will update account data by id.
func (ar *accountRepository) Update(ctx context.Context, id int, newData models.UpdateAccountIn) (err error) {
	var (
		values []interface{}
		query  = queryUpdate
	)

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	values = append(values, newData.IsHVT)

	if newData.Status != "" {
		query += `"status" = ?,`
		values = append(values, newData.Status)
	}
	query += ` "updatedAt" = now()`
	query += ` WHERE "id" = ?;`
	query = ar.r.SubstitutePlaceholder(query, 1)
	values = append(values, id)

	res, err := db.ExecContext(ctx, query, values...)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		err = common.ErrNoRowsAffected
		return
	}
	return
}

func (ar *accountRepository) Delete(ctx context.Context, accountID int) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryAccountDelete, accountID)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		err = common.ErrNoRowsAffected
		return
	}

	return
}

func (ar *accountRepository) UpdateBySubCategory(ctx context.Context, in models.UpdateAccountBySubCategoryIn) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	query, args, err := buildUpdateBySubCategoryQuery(in)
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	db := ar.r.extractTxWrite(ctx)
	_, err = db.ExecContext(ctx, query, args...)

	return
}

func (ar *accountRepository) DeleteByAccountNumber(ctx context.Context, accountNumber string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryDeleteAccountByAccountNumber, accountNumber)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		err = common.ErrNoRowsAffected
		return
	}

	return
}

// GetOneByAccountNumber will search account by it's account number on database.
func (ar *accountRepository) GetOneByAccountNumberOrLegacyId(ctx context.Context, accountNumber string) (result models.GetAccountOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := ar.r.extractTxWrite(ctx)

	var (
		actualBalance, pendingBalance                          decimal.Decimal
		balanceRangeMin, balanceRangeMax, negativeBalanceLimit decimal.NullDecimal
		negativeBalanceAllowed                                 sql.NullBool
		featurePreset                                          sql.NullString
	)

	result.Features = &models.WalletFeature{}

	err = db.QueryRowContext(ctx, queryGetOneByAccountNumberOrLegacyId, accountNumber).Scan(
		&result.ID,
		&result.AccountNumber,
		&result.OwnerID,
		&result.Category,
		&result.SubCategory,
		&result.Entity,
		&result.Currency,
		&result.Status,
		&result.IsHVT,
		&actualBalance,
		&pendingBalance,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.LegacyId,
		&result.AccountName,
		&featurePreset,
		&balanceRangeMin,
		&balanceRangeMax,
		&negativeBalanceAllowed,
		&negativeBalanceLimit,
	)
	if err != nil {
		return
	}

	preset := models.DefaultPresetWalletFeature
	if featurePreset.Valid {
		preset = featurePreset.String
	}
	upperCasePreset := strings.ToUpper(preset)

	defaultFeature, ok := ar.r.config.AccountFeatureConfig[preset]
	if !ok {
		return result, fmt.Errorf("preset not found: %s", featurePreset.String)
	}
	defaultNegativeBalanceAllowed := defaultFeature.NegativeBalanceAllowed
	defaultNegativeBalanceLimit := decimal.NewFromFloat(defaultFeature.NegativeLimit)
	defaultBalanceRangeMin := decimal.NewFromFloat(defaultFeature.BalanceRangeMin)
	defaultBalanceRangeMax := decimal.NewFromFloat(defaultFeature.BalanceRangeMax)

	if negativeBalanceAllowed.Valid {
		result.Features.AllowedNegativeBalance = &negativeBalanceAllowed.Bool
	} else {
		result.Features.AllowedNegativeBalance = &defaultNegativeBalanceAllowed
	}

	if negativeBalanceLimit.Valid {
		result.Features.NegativeBalanceLimit = &negativeBalanceLimit.Decimal
	} else {
		result.Features.NegativeBalanceLimit = &defaultNegativeBalanceLimit
	}

	if balanceRangeMin.Valid {
		result.Features.BalanceRangeMin = &balanceRangeMin.Decimal
	} else {
		result.Features.BalanceRangeMin = &defaultBalanceRangeMin
	}

	if balanceRangeMax.Valid {
		result.Features.BalanceRangeMax = &balanceRangeMax.Decimal
	} else {
		result.Features.BalanceRangeMax = &defaultBalanceRangeMax
	}

	result.Balance = models.NewBalance(actualBalance, pendingBalance, models.WithBalanceLimitEnabled(ar.r.flag.IsEnabled(ar.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)))
	result.Features.Preset = &upperCasePreset

	return
}

func (ar *accountRepository) GetCachedAccount(ctx context.Context, accountNumber string) (models.GetAccountOut, error) {
	return ar.r.cacheAccount.GetOrSet(ctx, cache.GetOrSetOpts[models.GetAccountOut]{
		Key: accountNumber,
		TTL: 12 * time.Hour,
		Callback: func() (models.GetAccountOut, error) {
			res, err := ar.GetOneByAccountNumberOrLegacyId(ctx, accountNumber)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return res, fmt.Errorf("%w: %s", common.ErrAccountNotExists, accountNumber)
				}

				return res, err
			}

			return res, nil
		},
	})
}
