package models

import (
	"context"
	"fmt"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/shopspring/decimal"
)

type WalletTransactionSet struct {
	FromAccount     string
	ToAccount       string
	TransactionType string
	Amount          decimal.Decimal
}

func NewWalletTransactionSet(fromAccount, toAccount string, amount decimal.Decimal, transactionType string) WalletTransactionSet {
	return WalletTransactionSet{
		FromAccount:     fromAccount,
		ToAccount:       toAccount,
		Amount:          amount,
		TransactionType: transactionType,
	}
}

// CalculateCashIn will increase ToAccount actualAmount
func (trx WalletTransactionSet) CalculateCashIn(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Withdraw(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to withdraw source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("to account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to add funds to destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destinationBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

// CalculateCashInCommit will calculate committed transaction from reserved transaction
func (trx WalletTransactionSet) CalculateCashInCommit(_ context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	// do nothing, since when isReserved is true, the actual balance is already increased
	return accountBalance, nil
}

// CalculateCashInCancel will decrease fromAccount actualBalance
func (trx WalletTransactionSet) CalculateCashInCancel(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	destinationBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.FromAccount)
	}

	err := destinationBalance.Withdraw(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to withdraw destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destinationBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = destinationBalance
	return accountBalance, nil
}

// CalculateCashOut will decrease fromAccount actualBalance
func (trx WalletTransactionSet) CalculateCashOut(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Withdraw(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to withdraw source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to add funds to destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destinationBalance),
		)
		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

// CalculateCashOutReserve will increase fromAccount pendingBalance
func (trx WalletTransactionSet) CalculateCashOutReserve(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Reserve(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to reserve source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance
	return accountBalance, nil
}

func (trx WalletTransactionSet) CalculateCashOutCommit(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Commit(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to commit source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to add funds to destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destinationBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

func (trx WalletTransactionSet) CalculateCashOutCancel(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.CancelReservation(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to cancel reserve source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance
	return accountBalance, nil
}

// CalculateTransfer will decrease fromAccount actualBalance
func (trx WalletTransactionSet) CalculateTransfer(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Withdraw(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to withdraw source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to add funds to destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destinationBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

func (trx WalletTransactionSet) CalculateTransferReserve(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Reserve(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to reserve source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	return accountBalance, nil
}

func (trx WalletTransactionSet) CalculateTransferCommit(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Commit(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to commit source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destBalance.AddFunds(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to add funds to destination account",
			xlog.String("accountNumber", trx.ToAccount),
			xlog.Any("balance", destBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destBalance

	return accountBalance, nil
}

func (trx WalletTransactionSet) CalculateTransferCancel(ctx context.Context, accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.CancelReservation(trx.Amount, WithTransactionType(trx.TransactionType))
	if err != nil {
		xlog.Error(ctx, "failed to cancel reserve source account",
			xlog.String("accountNumber", trx.FromAccount),
			xlog.Any("balance", sourceBalance),
		)

		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	return accountBalance, nil
}
