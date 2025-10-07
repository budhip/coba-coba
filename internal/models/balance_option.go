package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// BalanceOption is an option for creating a new Balance
type BalanceOption func(config *Balance)

// WithIgnoreBalanceSufficiency is used to bypass balance sufficiency checks
// this is used when transaction Acuan already happened from BU (ex: transaction from kafka message),
// and we just want to update the balance
func WithIgnoreBalanceSufficiency() BalanceOption {
	return func(c *Balance) {
		c.ignoreBalanceSufficiency = true
	}
}

// WithAllowedNegativeBalanceTransactionTypes is used to set the allowed negative balance transaction types
// this is used in negativeBalanceLimit check, if the transaction type is in this list, it will allow balance to go negative
func WithAllowedNegativeBalanceTransactionTypes(transactionTypes []string) BalanceOption {
	return func(c *Balance) {
		c.allowedNegativeBalanceTransactionTypes = transactionTypes
	}
}

// WithNegativeBalanceLimit is used to set the negative balance limit
// if the calculation balance reaches this limit, the transaction will be rejected
func WithNegativeBalanceLimit(negativeBalanceLimit decimal.Decimal) BalanceOption {
	return func(c *Balance) {
		c.negativeBalanceLimit = decimal.NewNullDecimal(negativeBalanceLimit)
	}
}

// WithHVT is used to indicate if the account is a High Volume Transaction account
func WithHVT() BalanceOption {
	return func(c *Balance) {
		c.isHVT = true
	}
}

// WithVersion is used to set the version of the balance
func WithVersion(version int) BalanceOption {
	return func(c *Balance) {
		c.version = version
	}
}

// WithLastUpdatedAt is used to set the last updated time of the balance
func WithLastUpdatedAt(lastUpdatedAt time.Time) BalanceOption {
	return func(c *Balance) {
		c.lastUpdatedAt = lastUpdatedAt
	}
}

// WithSkipBalanceUpdateOnDB is used to skip updating balance on DB
func WithSkipBalanceUpdateOnDB() BalanceOption {
	return func(c *Balance) {
		c.isSkipBalanceUpdateOnDB = true
	}
}

// WithBalanceLimitEnabled is used to skip balance limit validation
func WithBalanceLimitEnabled(balanceLimitToggle bool) BalanceOption {
	return func(c *Balance) {
		c.isBalanceLimitEnabled = balanceLimitToggle
	}
}

func WithBalanceRangeMax(balanceRangeMax decimal.Decimal) BalanceOption {
	return func(c *Balance) {
		c.balanceRangeMax = decimal.NewNullDecimal(balanceRangeMax)
	}
}

// calculateBalanceOption is an option for calculating balance
// this option for calculating Balance without modifying Balance struct
// this option is used as args in Reserve, CancelReservation, etc.
// add more options here if needed. for example: orderType, custom validation, bypass validation, etc
type calculateBalanceOption struct {
	transactionType string
}

func newCalculateBalanceOption(options ...CalculateBalanceOption) calculateBalanceOption {
	c := calculateBalanceOption{}
	for _, option := range options {
		option(&c)
	}
	return c
}

type CalculateBalanceOption func(*calculateBalanceOption)

func WithTransactionType(transactionType string) CalculateBalanceOption {
	return func(c *calculateBalanceOption) {
		c.transactionType = transactionType
	}
}
