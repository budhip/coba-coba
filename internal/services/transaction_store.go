package services

import (
	"context"
	"fmt"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/hashicorp/go-multierror"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func (ts *transaction) NewStoreBulkTransaction(ctx context.Context, req []models.TransactionReq) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	reqTransaction, accountNumbers, transactionSet, err := ts.transformAndValidate(ctx, req)
	if err != nil {
		return err
	}

	if len(reqTransaction) == 0 {
		// reqTransaction[].refNumber already exists
		return common.ErrOrderAlreadyExists
	}

	// make sure account exists before processing transaction
	if err = ts.ensureAccountExists(ctx, accountNumbers...); err != nil {
		return err
	}

	flagExcludeTransaction := ts.srv.conf.FeatureFlagKeyLookup.ExcludeConsumeTransactionFromSpecificSubCategory
	if ts.srv.flag.IsEnabled(flagExcludeTransaction) {
		variant, errGetVariant := flag.GetVariant[models.ExcludeConsumeTransactionVariant](ts.srv.flag, flagExcludeTransaction)
		if errGetVariant != nil {
			return fmt.Errorf("failed to get variant for feature flag %s: %w", flagExcludeTransaction, errGetVariant)
		}

		accounts, errGetAccounts := ts.srv.sqlRepo.GetAccountRepository().GetAllByAccountNumbers(ctx, accountNumbers)
		if errGetAccounts != nil {
			return fmt.Errorf("failed to get accounts: %w", errGetAccounts)
		}

		for _, act := range accounts {
			if slices.Contains(variant.Value.SubCategories, act.SubCategoryCode) {
				return common.ErrOrderContainExcludeInsertDB
			}
		}
	}

	// start wrap transaction here
	err = ts.srv.sqlRepo.Atomic(ctx, func(atomicCtx context.Context, r repositories.SQLRepository) error {
		// get account balances & calculate
		accountBalance, errAtomic := ts.getAccountBalancesAndCalculate(atomicCtx, accountNumbers, transactionSet)
		if errAtomic != nil {
			return errAtomic
		}

		// insert transaction
		if errAtomic = r.GetTransactionRepository().StoreBulkTransaction(atomicCtx, reqTransaction); errAtomic != nil {
			return errAtomic
		}

		for _, accountNumber := range accountNumbers {
			balance := accountBalance[accountNumber]

			// skip balance update for HVT account or excluded account (usually system account)
			skipBalanceUpdate := (ts.srv.conf.FeatureFlag.EnableDelayBalanceUpdateOnHVTAccount && balance.IsHVT()) ||
				slices.Contains(ts.srv.conf.AccountConfig.ExcludedBalanceUpdateAccountNumbers, accountNumber)
			if skipBalanceUpdate {
				continue
			}

			if _, errAtomic = r.GetAccountRepository().UpdateAccountBalance(atomicCtx, accountNumber, balance); err != nil {
				return errAtomic
			}
		}

		return nil
	})
	// end wrap transaction here

	return
}

func (ts *transaction) transformAndValidate(ctx context.Context, req []models.TransactionReq) ([]*models.Transaction, []string, []models.TransactionSet, error) {
	var (
		reqTransaction    []*models.Transaction
		rawReqTransaction []*models.Transaction
		refNumbers        []string
		accountNumbers    []string
		transactionSet    []models.TransactionSet
		errs              *multierror.Error
	)

	tTypes, err := ts.srv.masterDataRepo.GetListTransactionTypeCode(ctx)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("unable to get transaction type from master data: %v", err))
	}

	oTypes, err := ts.srv.masterDataRepo.GetListOrderTypeCode(ctx)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("unable to get order type from master data: %v", err))
	}

	acceptedOrderType := append(ts.srv.conf.TransactionValidationConfig.AcceptedOrderType, oTypes...)
	acceptedTransactionType := append(ts.srv.conf.TransactionValidationConfig.AcceptedTransactionType, tTypes...)

	for _, val := range req {
		en, err := val.ToRequest()
		if err != nil {
			errs = multierror.Append(errs, err)
		}

		orderType := strings.ToUpper(en.OrderType)
		if ok := slices.Contains(acceptedOrderType, orderType); !ok {
			errs = multierror.Append(errs, fmt.Errorf("invalid order type: %v", orderType))
		}

		if ok := slices.Contains(acceptedTransactionType, en.TypeTransaction); !ok {
			errs = multierror.Append(errs, fmt.Errorf("invalid transaction type: %v", en.TypeTransaction))
		}

		rawReqTransaction = append(rawReqTransaction, &en)
		refNumbers = append(refNumbers, en.RefNumber)
	}
	if errs.ErrorOrNil() != nil {
		return reqTransaction, accountNumbers, transactionSet, errs.ErrorOrNil()
	}

	exists, err := ts.srv.sqlRepo.GetTransactionRepository().CheckRefNumbers(ctx, refNumbers...)
	if err != nil {
		errs = multierror.Append(errs, err)
		return reqTransaction, accountNumbers, transactionSet, errs.ErrorOrNil()
	}

	for _, reqTrx := range rawReqTransaction {
		if exists[reqTrx.RefNumber] {
			xlog.Warn(ctx, fmt.Sprintf("duplicate refNumber: %s. skipping", reqTrx.RefNumber))
			continue
		}

		reqTransaction = append(reqTransaction, reqTrx)
		accountNumbers = append(accountNumbers, reqTrx.FromAccount, reqTrx.ToAccount)
		transactionSet = append(transactionSet, models.TransactionSet{
			FromAccount: reqTrx.FromAccount,
			ToAccount:   reqTrx.ToAccount,
			Amount:      reqTrx.Amount.Decimal,
		})
	}

	return reqTransaction, accountNumbers, transactionSet, errs.ErrorOrNil()
}

// getAccountBalancesAndCalculate get account balance and calculate the new balance
// this function only used for consuming transaction from kafka
func (ts *transaction) getAccountBalancesAndCalculate(ctx context.Context, accountNumbers []string, transactionSet []models.TransactionSet) (map[string]models.Balance, error) {
	calculatedAccountBalance := make(map[string]models.Balance, 0)

	accountBalances, err := ts.srv.sqlRepo.GetAccountRepository().GetAccountBalances(ctx, models.GetAccountBalanceRequest{
		AccountNumbers: accountNumbers,
		ExcludeHVT:     ts.srv.conf.FeatureFlag.EnableDelayBalanceUpdateOnHVTAccount,
		ForUpdate:      true,
	})
	if err != nil {
		return calculatedAccountBalance, err
	}

	for _, accountNumber := range accountNumbers {
		if balance, ok := accountBalances[accountNumber]; ok {
			// ignore balance validation since it comes from kafka
			accountBalances[accountNumber] = models.NewBalance(
				balance.Actual(),
				balance.Pending(),
				models.WithIgnoreBalanceSufficiency(),
				models.WithBalanceLimitEnabled(ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.BalanceLimitToggle)),
			)
		} else {
			// account is HVT
			accountBalances[accountNumber] = models.NewBalance(
				decimal.Zero,
				decimal.Zero,
				models.WithIgnoreBalanceSufficiency(),
				models.WithHVT(),
				models.WithBalanceLimitEnabled(ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.BalanceLimitToggle)),
			)
		}
	}

	// calculate new balance
	for _, v := range transactionSet {
		cab, errCalculate := v.Calculate(accountBalances)
		if errCalculate != nil {
			return calculatedAccountBalance, fmt.Errorf("unable to calculate account balance: %v", errCalculate)
		}

		maps.Copy(calculatedAccountBalance, cab)
	}

	return calculatedAccountBalance, nil
}
