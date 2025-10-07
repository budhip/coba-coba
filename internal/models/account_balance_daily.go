package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountBalanceDaily struct {
	AccountNumber string
	Date          *time.Time
	Balance       decimal.Decimal
}

type BalanceReconDifference struct {
	AccountNumber   string
	ConsumerBalance string
	DatabaseBalance string
}

func (e *BalanceReconDifference) ToReconFormat() []string {
	return []string{
		e.AccountNumber,
		e.ConsumerBalance,
		e.DatabaseBalance,
	}
}
