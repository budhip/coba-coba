package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID              int
	AccountNumber   string
	OwnerID         string
	ActualBalance   decimal.Decimal
	PendingBalance  decimal.Decimal
	Name            string
	ProductTypeName string
	SubCategoryCode string
}

type AccountMetadata map[string]any

func (e *AccountMetadata) Scan(src interface{}) error {
	var raw []byte
	switch src := src.(type) {
	case string:
		raw = []byte(src)
	case []byte:
		raw = src
	default:
		return fmt.Errorf("type %T not supported by Scan", src)
	}

	return json.Unmarshal(raw, e)
}

func (e AccountMetadata) Value() (value driver.Value, err error) {
	return json.Marshal(e)
}

type AccountUpsert struct {
	AccountNumber   string
	Name            string
	OwnerID         string
	ProductTypeName string
	CategoryCode    string
	SubCategoryCode string
	EntityCode      string
	Currency        string
	AltID           string
	LegacyId        *AccountLegacyId
	IsHVT           bool
	Status          string
	Metadata        AccountMetadata
}

type AccountLegacyId map[string]interface{}

func (al *AccountLegacyId) Value() (driver.Value, error) {
	// Convert AccountLegacyId to a JSON string representation.
	jsonValue, err := json.Marshal(al)
	if err != nil {
		return nil, err
	}
	return jsonValue, nil
}

func (al *AccountLegacyId) Scan(value interface{}) error {
	// Ensure the input value is of []byte type.
	jsonValue, ok := value.([]byte)
	if !ok {
		return errors.New("invalid JSON data")
	}

	// Unmarshal the JSON data into the AccountLegacyId struct.
	if err := json.Unmarshal(jsonValue, al); err != nil {
		return err
	}
	return nil
}

type AccountFilterOptions struct {
	Search        string
	AccountNumber string
	AccountName   string
	OwnerID       string
	Limit         int
	SortBy        string
	Sort          string

	Cursor *AccountCursor
}

func (e AccountFilterOptions) GetReversedSortDirection() string {
	if e.Sort == "asc" {
		return "desc"
	} else if e.Sort == "desc" {
		return "asc"
	} else {
		return ""
	}
}

type GetAccountBalanceRequest struct {
	AccountNumbers []string

	// ExcludeHVT is a flag to exclude HVT account from the result.
	// This is used to prevent HVT account from being included in the account balance calculation.
	ExcludeHVT bool

	// ForUpdate is a flag to lock the account balance for update in atomic transaction.
	ForUpdate bool

	// AccountNumbersExcludedFromDB is a list of account numbers to skip getting balance from DB.
	// it will automatically set the balance to 0 with ignoring balance sufficiency check on create transaction.
	// This is used to prevent HVT account from being included in the account balance calculation.
	// This options only available on BalanceRepository.GetMany
	AccountNumbersExcludedFromDB []string

	// OverrideBalanceOpts is a list of options to override the account balance.
	// This used in BalanceRepository.GetMany
	OverrideBalanceOpts []BalanceOption
}
