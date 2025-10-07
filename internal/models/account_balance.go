package models

import (
	"database/sql"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	AccountNumber    string
	T24AccountNumber string
	Balance          Balance
}

func (a *AccountBalance) ToModelResponse() DoGetAccountBalanceResponse {
	lastUpdate := common.FormatDatetimeToString(a.Balance.lastUpdatedAt.In(common.GetLocation()), common.DateFormatYYYYMMDDWithTimeAndOffset)

	return DoGetAccountBalanceResponse{
		Kind:             "accountBalance",
		AccountNumber:    a.AccountNumber,
		Currency:         IDRCurrency,
		ActualBalance:    a.Balance.Actual().String(),
		PendingBalance:   a.Balance.Pending().String(),
		AvailableBalance: a.Balance.Available().String(),
		LastUpdatedAt:    lastUpdate,
	}
}

func ConvertToBalanceMap(accountBalance []AccountBalance) (res map[string]Balance) {
	res = make(map[string]Balance)
	for _, v := range accountBalance {
		res[v.AccountNumber] = v.Balance
	}
	return
}

// AccountBalanceFeature is a struct that contains the account balance feature.
// this only used for getting data from SQL.
type AccountBalanceFeature struct {
	AccountNumber    string
	T24AccountNumber string
	Actual           decimal.Decimal
	Pending          decimal.Decimal
	IsHVT            sql.NullBool
	Version          sql.NullInt64
	LastUpdatedAt    time.Time

	Preset                 sql.NullString
	AllowedNegativeBalance sql.NullBool
	BalanceRangeMin        decimal.NullDecimal
	NegativeBalanceLimit   decimal.NullDecimal
	BalanceRangeMax        decimal.NullDecimal
}
