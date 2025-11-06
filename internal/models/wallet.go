package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/shopspring/decimal"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

const DefaultPresetWalletFeature = "customer"

// DTO Create Wallet Feature
type CreateWalletReq struct {
	AccountNumber string           `json:"-" param:"accountNumber" example:"21100100000001" validate:"required"`
	Features      WalletFeatureReq `json:"features"`
}

// DTO Create Feature
type WalletFeatureReq struct {
	Preset                 string `json:"preset"`
	AllowedNegativeBalance *bool  `json:"allowedNegativeBalance"`
	BalanceRangeMin        string `json:"balanceRangeMin"`
	BalanceRangeMax        string `json:"balanceRangeMax"`
	NegativeBalanceLimit   string `json:"negativeBalanceLimit"`
}

func (in *WalletFeatureReq) TransformAndValidate() (out WalletFeature, err error) {
	var (
		parseErr                                               error
		balanceRangeMin, balanceRangeMax, negativeBalanceLimit *decimal.Decimal
	)
	if in.BalanceRangeMin != "" {
		balanceRangeMin, err = common.NewDecimalFromString(in.BalanceRangeMin)
		if err != nil {
			parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse balanceRangeMin: %s", err.Error()))
		} else {
			if balanceRangeMin.LessThan(decimal.Zero) {
				parseErr = multierror.Append(parseErr, fmt.Errorf("balanceRangeMin must be greater or equal than 0"))
			}
		}
	}

	if in.BalanceRangeMax != "" {
		balanceRangeMax, err = common.NewDecimalFromString(in.BalanceRangeMax)
		if err != nil {
			parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse balanceRangeMax: %s", err.Error()))
		} else {
			if balanceRangeMax.LessThan(decimal.Zero) {
				parseErr = multierror.Append(parseErr, fmt.Errorf("balanceRangeMax must be greater or equal than 0"))
			}
		}
	}

	if in.NegativeBalanceLimit != "" {
		negativeBalanceLimit, err = common.NewDecimalFromString(in.NegativeBalanceLimit)
		if err != nil {
			parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse negativeBalanceLimit: %s", err.Error()))
		} else {
			if negativeBalanceLimit.LessThan(decimal.Zero) {
				parseErr = multierror.Append(parseErr, fmt.Errorf("negativeBalanceLimit must be greater or equal than 0"))
			}
		}
	}

	if parseErr != nil {
		return out, parseErr
	}

	lowPreset := strings.ToLower(in.Preset)
	out = WalletFeature{
		Preset:                 &lowPreset,
		BalanceRangeMin:        balanceRangeMin,
		BalanceRangeMax:        balanceRangeMax,
		NegativeBalanceLimit:   negativeBalanceLimit,
		AllowedNegativeBalance: in.AllowedNegativeBalance,
	}
	return
}

type CreateWalletIn struct {
	AccountNumber string
	Feature       *WalletFeature
}

type WalletFeature struct {
	Preset                 *string          `json:"preset,omitempty"`
	AllowedNegativeBalance *bool            `json:"allowedNegativeBalance,omitempty"`
	BalanceRangeMin        *decimal.Decimal `json:"balanceRangeMin"`
	BalanceRangeMax        *decimal.Decimal `json:"balanceRangeMax"`
	NegativeBalanceLimit   *decimal.Decimal `json:"negativeBalanceLimit,omitempty"`
}

type WalletOut struct {
	AccountNumber string         `json:"accountNumber"`
	Status        string         `json:"status"`
	Feature       *WalletFeature `json:"feature"`
	UpdatedDate   time.Time      `json:"updatedDate"`
	CreatedDate   time.Time      `json:"createdDate"`
}

type MapAccountFeature map[string]WalletFeature

func (in *CreateWalletReq) TransformAndValidate() (CreateWalletIn, error) {
	var (
		out      CreateWalletIn
		parseErr error
		err      error
	)
	balanceRangeMin, err := common.NewDecimalFromString(in.Features.BalanceRangeMin)
	if err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse balanceRangeMin: %s", err.Error()))
	}
	balanceRangeMax, err := common.NewDecimalFromString(in.Features.BalanceRangeMax)
	if err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse balanceRangeMax: %s", err.Error()))
	}
	negativeBalanceLimit, err := common.NewDecimalFromString(in.Features.NegativeBalanceLimit)
	if err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse negativeBalanceLimit: %s", err.Error()))
	}
	if parseErr != nil {
		return out, parseErr
	}
	lowPreset := strings.ToLower(in.Features.Preset)
	out = CreateWalletIn{
		AccountNumber: in.AccountNumber,
		Feature: &WalletFeature{
			Preset:                 &lowPreset,
			BalanceRangeMin:        balanceRangeMin,
			BalanceRangeMax:        balanceRangeMax,
			NegativeBalanceLimit:   negativeBalanceLimit,
			AllowedNegativeBalance: in.Features.AllowedNegativeBalance,
		},
	}
	return out, nil
}

func (a WalletOut) ToModelResponse() WalletResponse {
	return WalletResponse{
		Kind:          "accountFeature",
		AccountNumber: a.AccountNumber,
		Feature:       a.Feature,
	}
}

type WalletResponse struct {
	Kind          string         `json:"kind"`
	AccountNumber string         `json:"accountNumber"`
	Feature       *WalletFeature `json:"features"`
}

type UpdateWalletIn struct {
	AccountNumber string
	Feature       WalletFeature
}
