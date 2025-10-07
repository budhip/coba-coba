package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type BalanceCalculation func(accountBalance map[string]Balance) (map[string]Balance, error)

type TransactionSet struct {
	FromAccount string
	ToAccount   string
	Amount      decimal.Decimal
}

func NewTransactionSet(fromAccount, toAccount string, amount decimal.Decimal) TransactionSet {
	return TransactionSet{
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		Amount:      amount,
	}
}

// Calculate will calculate the balance of each account and will return the new balance
func (trx TransactionSet) Calculate(accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Withdraw(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

func (trx TransactionSet) CalculateReserve(accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Reserve(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	return accountBalance, nil
}

// CalculateCommit will move the reserved balance fromAccount to toAccount
func (trx TransactionSet) CalculateCommit(accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.Commit(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	destinationBalance, ok := accountBalance[trx.ToAccount]
	if !ok {
		return accountBalance, fmt.Errorf("destination account not found: %s", trx.ToAccount)
	}

	err = destinationBalance.AddFunds(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.ToAccount] = destinationBalance

	return accountBalance, nil
}

// CalculateCancel will reset reserved balance
func (trx TransactionSet) CalculateCancel(accountBalance map[string]Balance) (map[string]Balance, error) {
	sourceBalance, ok := accountBalance[trx.FromAccount]
	if !ok {
		return accountBalance, fmt.Errorf("source account not found: %s", trx.FromAccount)
	}

	err := sourceBalance.CancelReservation(trx.Amount)
	if err != nil {
		return accountBalance, err
	}

	accountBalance[trx.FromAccount] = sourceBalance

	return accountBalance, nil
}
