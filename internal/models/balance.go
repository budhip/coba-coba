package models

import (
	"encoding/json"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

type Balance struct {
	actualBalance  decimal.Decimal
	pendingBalance decimal.Decimal

	version       int
	lastUpdatedAt time.Time

	ignoreBalanceSufficiency               bool
	isHVT                                  bool
	isSkipBalanceUpdateOnDB                bool
	isBalanceLimitEnabled                  bool
	negativeBalanceLimit                   decimal.NullDecimal
	allowedNegativeBalanceTransactionTypes []string
	balanceRangeMax                        decimal.NullDecimal
}

type UpdateBalanceHVTPayload struct {
	Kind                string `json:"kind"`
	WalletTransactionId string `json:"walletTransactionId"`
	RefNumber           string `json:"refNumber"`
	AccountNumber       string `json:"accountNumber"`
	UpdateAmount        Amount `json:"updateAmount"`
}

func CreateUpdateBalanceHVTPayload(acuanTransaction Transaction) (*UpdateBalanceHVTPayload, error) {
	return &UpdateBalanceHVTPayload{
		AccountNumber: acuanTransaction.ToAccount,
		UpdateAmount: Amount{
			NewDecimalFromExternal(acuanTransaction.Amount.Decimal),
			acuanTransaction.Currency},
	}, nil
}

func NewBalance(actualBalance, pendingBalance decimal.Decimal, options ...BalanceOption) Balance {
	b := Balance{
		actualBalance:  actualBalance,
		pendingBalance: pendingBalance,
	}

	for _, option := range options {
		option(&b)
	}

	return b
}

// Reserve temporarily increases the pending balance.
// This could be used when expecting a future transaction that isn't confirmed yet.
func (b *Balance) Reserve(amount decimal.Decimal, opt ...CalculateBalanceOption) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return common.ErrInvalidAmount
	}

	cbo := newCalculateBalanceOption(opt...)

	if !b.ignoreBalanceSufficiency {
		isTransactionTypeAllowedForNegativeBalance := cbo.transactionType != "" &&
			slices.Contains(b.allowedNegativeBalanceTransactionTypes, cbo.transactionType)

		// check if balance is have option to go negative
		if b.negativeBalanceLimit.Valid && isTransactionTypeAllowedForNegativeBalance {
			balanceLimit := decimal.Zero.Sub(b.negativeBalanceLimit.Decimal)
			negativeBalanceReached := b.Available().Sub(amount).LessThan(balanceLimit)

			if negativeBalanceReached {
				return common.ErrNegativeBalanceReached
			}
		} else if b.Available().LessThan(amount) {
			return common.ErrInsufficientAvailableBalance
		}
	}

	b.pendingBalance = b.pendingBalance.Add(amount)

	return nil
}

// CancelReservation reverses a previous reservation
func (b *Balance) CancelReservation(amount decimal.Decimal, _ ...CalculateBalanceOption) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return common.ErrInvalidAmount
	}

	if !b.ignoreBalanceSufficiency && b.Pending().LessThan(amount) {
		return common.ErrInsufficientPendingBalance
	}

	b.pendingBalance = b.pendingBalance.Sub(amount)

	return nil
}

// Commit finalizes a reservation by reducing both the actual balance and pending balance.
// This could represent a pending transaction becoming confirmed
func (b *Balance) Commit(amount decimal.Decimal, _ ...CalculateBalanceOption) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return common.ErrInvalidAmount
	}

	if !b.ignoreBalanceSufficiency && b.Pending().LessThan(amount) {
		return common.ErrInsufficientPendingBalance
	}

	b.actualBalance = b.actualBalance.Sub(amount)
	b.pendingBalance = b.pendingBalance.Sub(amount)

	return nil
}

// AddFunds increases the actual balance
func (b *Balance) AddFunds(amount decimal.Decimal, _ ...CalculateBalanceOption) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return common.ErrInvalidAmount
	}

	if b.balanceRangeMax.Valid && b.balanceRangeMax.Decimal.GreaterThan(decimal.Zero) {
		if b.actualBalance.Add(amount).GreaterThan(b.balanceRangeMax.Decimal) && b.isBalanceLimitEnabled {
			return common.ErrMaxBalanceExceeded
		}
	}

	b.actualBalance = b.actualBalance.Add(amount)

	return nil
}

// Withdraw decreases the actual balance
func (b *Balance) Withdraw(amount decimal.Decimal, opt ...CalculateBalanceOption) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return common.ErrInvalidAmount
	}

	cbo := newCalculateBalanceOption(opt...)

	if !b.ignoreBalanceSufficiency {
		isTransactionTypeAllowedForNegativeBalance := cbo.transactionType != "" &&
			slices.Contains(b.allowedNegativeBalanceTransactionTypes, cbo.transactionType)

		// check if balance is have option to go negative
		if b.negativeBalanceLimit.Valid && isTransactionTypeAllowedForNegativeBalance {
			balanceLimit := decimal.Zero.Sub(b.negativeBalanceLimit.Decimal)
			negativeBalanceReached := b.Available().Sub(amount).LessThan(balanceLimit)

			if negativeBalanceReached {
				return common.ErrNegativeBalanceReached
			}
		} else if b.Available().LessThan(amount) {
			return common.ErrInsufficientAvailableBalance
		}
	}

	b.actualBalance = b.actualBalance.Sub(amount)

	return nil
}

func (b *Balance) Available() decimal.Decimal {
	return b.actualBalance.Sub(b.pendingBalance)
}

func (b *Balance) Actual() decimal.Decimal {
	return b.actualBalance
}

func (b *Balance) Pending() decimal.Decimal {
	return b.pendingBalance
}

func (b *Balance) IsHVT() bool {
	return b.isHVT
}

func (b *Balance) NegativeBalanceLimit() decimal.NullDecimal {
	return b.negativeBalanceLimit
}

func (b *Balance) IsSkipBalanceUpdateOnDB() bool {
	return b.isSkipBalanceUpdateOnDB
}

// balanceJSON is used to marshal/unmarshal Balance to/from JSON
// this is needed because Balance is a struct with unexported fields
// we use private fields to prevent direct access to the Balance fields
// so, we can control the balance changes through the Balance methods
type balanceJSON struct {
	ActualBalance  decimal.Decimal `json:"actualBalance"`
	PendingBalance decimal.Decimal `json:"pendingBalance"`
}

func (b Balance) MarshalJSON() ([]byte, error) {
	return json.Marshal(balanceJSON{
		ActualBalance:  b.actualBalance,
		PendingBalance: b.pendingBalance,
	})
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var jsonBalance balanceJSON

	if err := json.Unmarshal(data, &jsonBalance); err != nil {
		return err
	}

	b.actualBalance = jsonBalance.ActualBalance
	b.pendingBalance = jsonBalance.PendingBalance

	return nil
}

// balanceJSONV2 is new version of Balance JSON
// this is used for send notification to Acuan while preserving the old Balance JSON
// that used in get balance API
type balanceJSONV2 struct {
	ActualBalance    Amount    `json:"actualBalance"`
	PendingBalance   Amount    `json:"pendingBalance"`
	AvailableBalance Amount    `json:"availableBalance"`
	Version          int       `json:"version"`
	LastUpdatedAt    time.Time `json:"lastUpdatedAt"`
}

func newBalanceJSONV2(b Balance) balanceJSONV2 {
	return balanceJSONV2{
		ActualBalance: Amount{
			ValueDecimal: NewDecimalFromExternal(b.actualBalance),
			Currency:     IDRCurrency,
		},
		PendingBalance: Amount{
			ValueDecimal: NewDecimalFromExternal(b.pendingBalance),
			Currency:     IDRCurrency,
		},
		AvailableBalance: Amount{
			ValueDecimal: NewDecimalFromExternal(b.Available()),
			Currency:     IDRCurrency,
		},
		Version:       b.version,
		LastUpdatedAt: b.lastUpdatedAt,
	}
}
