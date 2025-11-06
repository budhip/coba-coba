package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"golang.org/x/exp/slices"
)

func checkDatabaseError(err error, code ...string) error {
	if errors.Is(err, common.ErrNoRows) {
		err = models.GetErrMap(models.ErrKeyDataNotFound)
		if len(code) > 0 {
			err = models.GetErrMap(code[0])
		}
	} else {
		err = models.GetErrMap(models.ErrKeyDatabaseError, err.Error())
	}

	return err
}

type balanceCalculator func(trxSet models.TransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error)

func getBalanceCalculator(processType models.TransactionStoreProcessType) balanceCalculator {
	if processType == models.TransactionStoreProcessReserved {
		return func(trxSet models.TransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateReserve(accountBalance)
		}
	}

	return func(trxSet models.TransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
		return trxSet.Calculate(accountBalance)
	}
}

type walletBalanceCalculator func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error)

func getWalletBalanceCalculator(trxFlow models.TransactionFlow, isReserved bool) walletBalanceCalculator {
	switch trxFlow {
	case models.TransactionFlowCashIn:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashIn(ctx, accountBalance)
		}
	case models.TransactionFlowCashOut:
		if isReserved {
			return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
				return trxSet.CalculateCashOutReserve(ctx, accountBalance)
			}
		}

		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashOut(ctx, accountBalance)
		}
	case models.TransactionFlowTransfer, models.TransactionFlowRefund:
		if isReserved {
			return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
				return trxSet.CalculateTransferReserve(ctx, accountBalance)
			}
		}

		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateTransfer(ctx, accountBalance)
		}
	default:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return nil, fmt.Errorf("transactionFlow not supported: %s", trxFlow)
		}
	}
}

func getWalletBalanceCommitCalculator(trxFlow models.TransactionFlow) walletBalanceCalculator {
	switch trxFlow {
	case models.TransactionFlowCashIn:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashInCommit(ctx, accountBalance)
		}
	case models.TransactionFlowCashOut:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashOutCommit(ctx, accountBalance)
		}
	case models.TransactionFlowTransfer, models.TransactionFlowRefund:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateTransferCommit(ctx, accountBalance)
		}
	default:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return nil, fmt.Errorf("transactionFlow not supported: %s", trxFlow)
		}
	}
}

func getWalletBalanceCancelCalculator(trxFlow models.TransactionFlow) walletBalanceCalculator {
	switch trxFlow {
	case models.TransactionFlowCashIn:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashInCancel(ctx, accountBalance)
		}
	case models.TransactionFlowCashOut:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateCashOutCancel(ctx, accountBalance)
		}
	case models.TransactionFlowTransfer, models.TransactionFlowRefund:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return trxSet.CalculateTransferCancel(ctx, accountBalance)
		}
	default:
		return func(ctx context.Context, trxSet models.WalletTransactionSet, accountBalance map[string]models.Balance) (map[string]models.Balance, error) {
			return nil, fmt.Errorf("transactionFlow not supported: %s", trxFlow)
		}
	}
}

func getCacheKeyAccountAndBalance(accountNumbers ...string) []string {
	var cacheKeys []string
	for _, accountNumber := range accountNumbers {
		cacheKeys = append(cacheKeys, fmt.Sprintf("fp:account-balance:%s", accountNumber))
	}

	return cacheKeys
}

func getAccountNumbersForUpdateBalance(acuanTransactions []models.TransactionReq) []string {
	var accountNumbers []string
	for _, acuanTransaction := range acuanTransactions {
		if acuanTransaction.FromAccount != "" {
			accountNumbers = append(accountNumbers, acuanTransaction.FromAccount)
		}

		if acuanTransaction.ToAccount != "" {
			accountNumbers = append(accountNumbers, acuanTransaction.ToAccount)
		}
	}

	return accountNumbers
}

// getIgnoreBalanceCheckAccountNumbers returns account numbers that should be ignored from balance check
// usually it is system account or account number that can be negative
func getIgnoreBalanceCheckAccountNumbers(cfg config.Config) []string {
	accountNumbers := append(
		[]string{
			cfg.AccountConfig.SystemAccountNumber,
			cfg.AccountConfig.BPE,
			cfg.AccountConfig.BRIEscrowAFAAccountNumber,
		},
		cfg.TransactionValidationConfig.SkipBalanceCheckAccountNumber...,
	)

	for _, an := range cfg.AccountConfig.OperationalReceivableAccountNumberByEntity {
		accountNumbers = append(accountNumbers, an)
	}

	return accountNumbers
}

// walletTransactionNetAmountCalculator is contract to calculate net amount.
type walletTransactionNetAmountCalculator func(trx models.WalletTransaction) models.Amount

// mapWalletTransactionAmounts is function to map the wallet transaction amounts array into map with Type as key.
func mapWalletTransactionAmounts(amounts models.Amounts) map[string]models.Amount {
	mapAmount := make(map[string]models.Amount)
	for _, amount := range amounts {
		mapAmount[amount.Type] = *amount.Amount
	}
	return mapAmount
}

// getWalletTransactionNetAmountCalculator is function to get the walletTransactionNetAmountCalculator
// most cases it will be based on the trx.TransactionType
// the default calculator will return the trx.NetAmount.
func getWalletTransactionNetAmountCalculator(trx models.WalletTransaction) walletTransactionNetAmountCalculator {
	// simple version for now, if the case growing, need to move to proper mapping.
	switch trx.TransactionType {
	case "RPYAD":
		return func(trx models.WalletTransaction) models.Amount {
			amounts := mapWalletTransactionAmounts(trx.Amounts)
			var rpyab models.Amount
			if amount, ok := amounts["RPYAB"]; ok {
				rpyab = amount
			} else {
				zeroDec, _ := models.NewDecimal("0")
				rpyab = models.Amount{
					ValueDecimal: zeroDec,
				}
			}
			return models.Amount{
				ValueDecimal: models.NewDecimalFromExternal(trx.NetAmount.ValueDecimal.Add(rpyab.ValueDecimal.Decimal)),
				Currency:     trx.NetAmount.Currency,
			}
		}
	case "COTLR":
		return func(trx models.WalletTransaction) models.Amount {
			amounts := mapWalletTransactionAmounts(trx.Amounts)
			var admce models.Amount
			if amount, ok := amounts["ADMCE"]; ok {
				admce = amount
			} else {
				zeroDec, _ := models.NewDecimal("0")
				admce = models.Amount{
					ValueDecimal: zeroDec,
				}
			}
			return models.Amount{
				ValueDecimal: models.NewDecimalFromExternal(trx.NetAmount.ValueDecimal.Add(admce.ValueDecimal.Decimal)),
				Currency:     trx.NetAmount.Currency,
			}
		}
	case "FPEPD":
		return func(trx models.WalletTransaction) models.Amount {
			amounts := mapWalletTransactionAmounts(trx.Amounts)
			var itded models.Amount
			if amount, ok := amounts["ITDED"]; ok {
				itded = amount
			} else {
				zeroDec, _ := models.NewDecimal("0")
				itded = models.Amount{
					ValueDecimal: zeroDec,
				}
			}
			return models.Amount{
				ValueDecimal: models.NewDecimalFromExternal(trx.NetAmount.ValueDecimal.Add(itded.ValueDecimal.Decimal)),
				Currency:     trx.NetAmount.Currency,
			}
		}
	default:
	}
	return func(trx models.WalletTransaction) models.Amount {
		return trx.NetAmount
	}
}

func calculateTotalAmountOfTransactions(oldTransaction []models.WalletTransaction) []models.WalletTransaction {
	newTransaction := []models.WalletTransaction{}

	for _, trx := range oldTransaction {
		calculator := getWalletTransactionNetAmountCalculator(trx)
		trx.NetAmount = calculator(trx)
		newTransaction = append(newTransaction, trx)
	}

	return newTransaction
}

func validateAccountExistsInTransactions(trxReq []models.TransactionReq, balances []models.AccountBalance) error {
	var accountNumbers []string
	for _, b := range balances {
		accountNumbers = append(accountNumbers, b.AccountNumber)
		if b.T24AccountNumber != "" {
			accountNumbers = append(accountNumbers, b.T24AccountNumber)
		}
	}

	for _, req := range trxReq {
		if !slices.Contains(accountNumbers, req.FromAccount) {
			return fmt.Errorf("%w: %s", common.ErrAccountNotExists, req.FromAccount)
		}

		if !slices.Contains(accountNumbers, req.ToAccount) {
			return fmt.Errorf("%w: %s", common.ErrAccountNotExists, req.ToAccount)
		}
	}

	return nil
}

func mapT24AccountNumberToAccountNumber(balances []models.AccountBalance) map[string]string {
	mapAccountNumber := make(map[string]string)

	for _, b := range balances {
		if b.T24AccountNumber != "" {
			mapAccountNumber[b.T24AccountNumber] = b.AccountNumber
		}

		mapAccountNumber[b.AccountNumber] = b.AccountNumber
	}

	return mapAccountNumber
}

// updateTransactionAccountNumber is a function to update the account number if t24 account number is used in
// transaction request
func updateTransactionAccountNumber(trxReq []models.TransactionReq, balances []models.AccountBalance) []models.TransactionReq {
	mapAccountNumber := mapT24AccountNumberToAccountNumber(balances)

	for i, req := range trxReq {
		if req.FromAccount != "" {
			if val, ok := mapAccountNumber[req.FromAccount]; ok {
				trxReq[i].FromAccount = val
			}
		}
		if req.ToAccount != "" {
			if val, ok := mapAccountNumber[req.ToAccount]; ok {
				trxReq[i].ToAccount = val
			}
		}
	}

	return trxReq
}

func getCacheKeyReportRepayment(startDate, endDate time.Time) string {
	return fmt.Sprintf("go-fp:report-repayment:%s:%s", startDate.Format(time.DateOnly), endDate.Format(time.DateOnly))
}
