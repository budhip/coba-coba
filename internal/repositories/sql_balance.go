package repositories

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

type BalanceRepository interface {
	Get(ctx context.Context, accountNumber string) (models.AccountBalance, error)
	GetMany(ctx context.Context, req models.GetAccountBalanceRequest) ([]models.AccountBalance, error)
	AdjustAccountBalance(ctx context.Context, accountNumber string, updatedAmount models.Decimal) error

	// add more method related to balance here. ex: Update, etc.
	// TODO: move GetAccountBalances, UpdateAccountBalance from AccountRepository to BalanceRepository
}

type balanceRepository sqlRepo

var _ BalanceRepository = (*balanceRepository)(nil)

func (b balanceRepository) Get(ctx context.Context, accountNumber string) (res models.AccountBalance, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := b.r.extractTxWrite(ctx)

	var abf models.AccountBalanceFeature
	err = db.
		QueryRowContext(ctx, queryGetAccountBalanceWithFeature, accountNumber).
		Scan(
			&abf.AccountNumber,
			&abf.T24AccountNumber,
			&abf.Actual,
			&abf.Pending,
			&abf.IsHVT,
			&abf.Version,
			&abf.LastUpdatedAt,
			&abf.Preset,
			&abf.AllowedNegativeBalance,
			&abf.BalanceRangeMin,
			&abf.NegativeBalanceLimit,
			&abf.BalanceRangeMax,
		)
	if err != nil {
		return res, err
	}

	ignoredAccounts, err := b.getIgnoredAccounts()
	if err != nil {
		return res, err
	}

	balanceOpts, err := createBalanceOptions(abf, ignoredAccounts, b.r.flag, b.r.config)
	if err != nil {
		return res, err
	}

	res = models.AccountBalance{
		AccountNumber:    abf.AccountNumber,
		T24AccountNumber: abf.T24AccountNumber,
		Balance:          models.NewBalance(abf.Actual, abf.Pending, balanceOpts...),
	}

	return res, nil
}

func (b balanceRepository) GetMany(ctx context.Context, req models.GetAccountBalanceRequest) (res []models.AccountBalance, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := b.r.extractTxWrite(ctx)

	ignoredAccounts, err := b.getIgnoredAccounts()
	if err != nil {
		return res, err
	}

	query, args, err := buildGetManyAccountBalanceQuery(req, ignoredAccounts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var abf models.AccountBalanceFeature
		err = rows.Scan(
			&abf.AccountNumber,
			&abf.T24AccountNumber,
			&abf.Actual,
			&abf.Pending,
			&abf.IsHVT,
			&abf.Version,
			&abf.LastUpdatedAt,
			&abf.Preset,
			&abf.AllowedNegativeBalance,
			&abf.BalanceRangeMin,
			&abf.NegativeBalanceLimit,
			&abf.BalanceRangeMax,
		)
		if err != nil {
			return nil, err
		}

		balanceOpts, errCreateOpts := createBalanceOptions(abf, ignoredAccounts, b.r.flag, b.r.config)
		if errCreateOpts != nil {
			return res, errCreateOpts
		}

		if len(req.OverrideBalanceOpts) > 0 {
			balanceOpts = append(balanceOpts, req.OverrideBalanceOpts...)
		}

		res = append(res, models.AccountBalance{
			AccountNumber:    abf.AccountNumber,
			T24AccountNumber: abf.T24AccountNumber,
			Balance:          models.NewBalance(abf.Actual, abf.Pending, balanceOpts...),
		})
	}

	for _, accountNumber := range req.AccountNumbers {
		if slices.Contains(req.AccountNumbersExcludedFromDB, accountNumber) || slices.Contains(ignoredAccounts, accountNumber) {
			res = append(res, models.AccountBalance{
				AccountNumber: accountNumber,
				Balance: models.NewBalance(
					decimal.Zero,
					decimal.Zero,
					models.WithIgnoreBalanceSufficiency(),
					models.WithSkipBalanceUpdateOnDB(),
					models.WithBalanceLimitEnabled(b.r.flag.IsEnabled(b.r.config.FeatureFlagKeyLookup.BalanceLimitToggle)),
				),
			})
		}
	}

	return res, nil
}

// getIgnoredAccounts returns a list of account numbers that should be ignored when checking balance.
func (b balanceRepository) getIgnoredAccounts() (accountNumbers []string, err error) {
	key := b.r.config.FeatureFlagKeyLookup.IgnoredBalanceCheckAccountNumbers
	variant, err := flag.GetVariant[[]string](b.r.flag, key)
	if err != nil {
		return nil, err
	}

	if variant.Enabled {
		accountNumbers = append(accountNumbers, variant.Value...)
	}

	accountNumbers = append(accountNumbers,
		b.r.config.AccountConfig.SystemAccountNumber,
		b.r.config.AccountConfig.BPE,
		b.r.config.AccountConfig.BRIEscrowAFAAccountNumber)

	for _, an := range b.r.config.TransactionValidationConfig.SkipBalanceCheckAccountNumber {
		accountNumbers = append(accountNumbers, an)
	}

	for _, an := range b.r.config.AccountConfig.OperationalReceivableAccountNumberByEntity {
		accountNumbers = append(accountNumbers, an)
	}

	return accountNumbers, nil
}

func (b balanceRepository) AdjustAccountBalance(ctx context.Context, accountNumber string, updateAmount models.Decimal) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := b.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryAdjustAccountBalance, updateAmount.Decimal, accountNumber)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		return common.ErrNoRowsAffected
	}

	return nil
}
