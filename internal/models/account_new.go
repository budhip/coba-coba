package models

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/shopspring/decimal"
)

var kindAccount = "account"

type CreateAccount struct {
	AccountNumber   string
	Name            string
	OwnerID         string
	ProductTypeName string
	CategoryCode    string
	SubCategoryCode string
	EntityCode      string
	Currency        string
	AltId           string
	LegacyId        *AccountLegacyId
	IsHVT           bool
	Status          string
	Metadata        AccountMetadata
}

func (a *CreateAccount) ToCreateAccountResponse() *DoCreateAccountResponse {
	return &DoCreateAccountResponse{
		Kind:            kindAccount,
		AccountNumber:   a.AccountNumber,
		Name:            a.Name,
		OwnerID:         a.OwnerID,
		CategoryCode:    a.CategoryCode,
		SubCategoryCode: a.SubCategoryCode,
		EntityCode:      a.EntityCode,
		Currency:        a.Currency,
		AltId:           a.AltId,
		LegacyId:        a.LegacyId,
		Status:          a.Status,
	}
}

type AccountsTotalBalanceResponse struct {
	Kind         string           `json:"kind" example:"account"`
	TotalBalance *decimal.Decimal `json:"totalBalance" example:"100.2"`
}

func NewAccountsTotalBalanceResponse(totalBalance *decimal.Decimal) *AccountsTotalBalanceResponse {
	return &AccountsTotalBalanceResponse{
		Kind:         "account",
		TotalBalance: totalBalance,
	}
}

type (
	DoCreateAccountRequest struct {
		AccountNumber   string           `json:"accountNumber" validate:"required,numeric" example:"21100100000001"`
		Name            string           `json:"name" validate:"required" example:"John"`
		OwnerID         string           `json:"ownerId" validate:"required,alphanum,min=1,max=15" example:"12345"`
		ProductTypeName string           `json:"productTypeName" example:"BroilerX"`
		CategoryCode    string           `json:"categoryCode" validate:"required,numeric,min=3,max=3" example:"211"`
		SubCategoryCode string           `json:"subCategoryCode" validate:"required,numeric,min=5,max=5" example:"10000"`
		EntityCode      string           `json:"entityCode" validate:"required,min=3,max=3,numeric" example:"001"`
		Currency        string           `json:"currency" validate:"required,alpha,min=3,max=3" example:"IDR"`
		AltId           string           `json:"altId" validate:"omitempty,min=1,max=50" example:"12345"`
		LegacyId        *AccountLegacyId `json:"legacyId" swaggertype:"object,string" validate:"omitempty" example:"t24AccountNumber:1234567890,t24ArrangementId:1234567890"`
		Metadata        map[string]any   `json:"metadata" swaggertype:"object,string" example:"key:value"`
		Status          string           `json:"status" validate:"required,oneof=active inactive" example:"active"`
	}
	DoCreateAccountResponse struct {
		Kind            string           `json:"kind" example:"account"`
		AccountNumber   string           `json:"accountNumber" example:"21100100000001"`
		Name            string           `json:"name" example:"John"`
		OwnerID         string           `json:"ownerId" example:"12345"`
		CategoryCode    string           `json:"categoryCode" example:"211"`
		SubCategoryCode string           `json:"subCategoryCode" example:"10000"`
		EntityCode      string           `json:"entityCode" example:"001"`
		Currency        string           `json:"currency" example:"IDR"`
		AltId           string           `json:"altId" example:"12345"`
		LegacyId        *AccountLegacyId `json:"legacyId" swaggertype:"object,string" example:"t24AccountNumber:1234567890,t24ArrangementId:1234567890"`
		Status          string           `json:"status" example:"active"`
	}
	DoPatchAccountRequest struct {
		Action string `json:"action" validate:"required,oneof=update_name" example:"update_name"`
	}
)

type GetAccountOut struct {
	ID            int
	AccountName   string
	AccountNumber string
	OwnerID       string
	Category      string
	SubCategory   string
	Entity        string
	Currency      string
	Status        string
	Balance       Balance
	IsHVT         bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LegacyId      *AccountLegacyId

	Features *WalletFeature
}

type GetAccountResponse struct {
	Kind             string         `json:"kind" example:"account"`
	OwnerID          string         `json:"ownerId" example:"5432"`
	AccountNumber    string         `json:"accountNumber" example:"21100100000001"`
	AccountName      string         `json:"accountName" example:"John"`
	Currency         string         `json:"currency" example:"IDR"`
	AvailableBalance string         `json:"availableBalance" example:"10000"`
	PendingBalance   string         `json:"pendingBalance" example:"10000"`
	ActualBalance    string         `json:"actualBalance" example:"10000"`
	Status           string         `json:"status" example:"active"`
	Features         *WalletFeature `json:"features"`
	CreatedAt        string         `json:"createdAt" example:"2006-01-02 15:04:05"`
	UpdatedAt        string         `json:"updatedAt" example:"2006-01-02 15:04:05"`
}

func (a GetAccountOut) GetCursor() string {
	ac := AccountCursor{
		Id:        a.ID,
		UpdatedAt: a.UpdatedAt,
	}

	return ac.Encode()
}

func (a GetAccountOut) ToModelResponse() GetAccountResponse {
	return GetAccountResponse{
		Kind:             kindAccount,
		OwnerID:          a.OwnerID,
		AccountName:      a.AccountName,
		AccountNumber:    a.AccountNumber,
		Currency:         a.Currency,
		Status:           a.Status,
		AvailableBalance: a.Balance.Available().String(),
		PendingBalance:   a.Balance.Pending().String(),
		ActualBalance:    a.Balance.Actual().String(),
		Features:         a.Features,
		CreatedAt:        a.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		UpdatedAt:        a.UpdatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
	}
}

func (a GetAccountOut) ToModelResponseGetBalance() DoGetAccountBalanceResponse {
	return DoGetAccountBalanceResponse{
		Kind:             "accountBalance",
		AccountNumber:    a.AccountNumber,
		Currency:         a.Currency,
		ActualBalance:    a.Balance.Actual().String(),
		PendingBalance:   a.Balance.Pending().String(),
		AvailableBalance: a.Balance.Available().String(),
		LastUpdatedAt:    common.FormatDatetimeToString(a.UpdatedAt.In(common.GetLocation()), common.DateFormatYYYYMMDDWithTimeAndOffset),
	}
}

type DoGetAccountRequest struct {
	AccountNumber string `params:"accountNumber" example:"21100100000001"`
}

type DoGetAccountBalanceResponse struct {
	Kind             string `json:"kind" example:"accountBalance"`
	AccountNumber    string `json:"accountNumber" example:"21100100000001"`
	Currency         string `json:"currency" example:"IDR"`
	ActualBalance    string `json:"actualBalance" example:"10000"`
	PendingBalance   string `json:"pendingBalance" example:"10000"`
	AvailableBalance string `json:"availableBalance" example:"10000"`
	LastUpdatedAt    string `json:"lastUpdatedAt" example:"2024-01-22T15:51:43+0700"` //ISO 8601
}

type DoGetListAccountRequest struct {
	Search        string `query:"search" example:"accountNumber"`
	AccountNumber string `query:"accountNumber" example:"213001000000033"`
	AccountName   string `query:"accountName" example:"Sweet Heart"`
	OwnerID       string `query:"ownerID" example:"Vxo2hruvBiOaBsE"`
	Limit         int    `query:"limit" example:"10"`
	SortBy        string `query:"sortBy" validate:"omitempty,oneof=lastUpdatedDate" example:"lastUpdatedDate"`
	Sort          string `query:"sort" validate:"omitempty,oneof=asc desc" example:"asc or desc"`
	NextCursor    string `query:"nextCursor" example:"abc"`
	PrevCursor    string `query:"prevCursor" example:"cba"`
}

func (req DoGetListAccountRequest) ToFilterOpts() (*AccountFilterOptions, error) {
	opts := &AccountFilterOptions{
		Search:        req.Search,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		OwnerID:       req.OwnerID,
		Limit:         req.Limit,
		SortBy:        req.SortBy,
		Sort:          req.Sort,
	}

	if req.Limit < 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	// default limit
	if req.Limit == 0 {
		opts.Limit = 10
	}

	// default sortBy
	if req.SortBy == "" {
		opts.SortBy = "createdAt"
	}

	// default sort direction
	if req.Sort == "" {
		opts.Sort = "desc"
	}

	// use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// forward pagination
	if req.NextCursor != "" {
		ac, err := decodeAccountCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}

		opts.Cursor = ac
	}

	// backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		ac, err := decodeAccountCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}

		ac.IsBackward = true
		opts.Cursor = ac
	}

	return opts, nil
}

type AccountCursor struct {
	Id         int
	UpdatedAt  time.Time
	IsBackward bool
}

func (ac AccountCursor) Encode() string {
	s := fmt.Sprintf("%d|%s", ac.Id, ac.UpdatedAt.Format(time.RFC3339Nano))
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// decodeAccountCursor will decode cursor into time.Time
func decodeAccountCursor(cursor string) (ac *AccountCursor, err error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset string: %w", err)
	}

	splitCursor := strings.Split(string(decodedBytes), "|")
	if len(splitCursor) != 2 {
		return nil, fmt.Errorf("failed to parse cursor: invalid format")
	}

	id, err := strconv.Atoi(splitCursor[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset id: %w", err)
	}

	decodedTime, err := time.Parse(time.RFC3339Nano, splitCursor[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset time: %w", err)
	}

	return &AccountCursor{
		Id:        id,
		UpdatedAt: decodedTime,
	}, nil
}

type UpdateAccountRequest struct {
	AccountNumber string            `json:"-" params:"accountNumber" example:"21100100000001" validate:"required"`
	IsHVT         *bool             `json:"isHvt" example:"true" validate:"required"`
	Status        string            `json:"status" example:"active"`
	Feature       *WalletFeatureReq `json:"features"`
}

func (in *UpdateAccountRequest) TransformAndValidate() (out UpdateAccountIn, err error) {
	// Checking status
	if in.Status != "" {
		if _, ok := common.MapAccountStatusReverse[in.Status]; !ok {
			err = fmt.Errorf("invalid status value: %s", in.Status)
			return
		}
	}

	feature := WalletFeature{}
	if in.Feature != nil {
		feature, err = in.Feature.TransformAndValidate()
		if err != nil {
			return
		}
	}

	out = UpdateAccountIn{
		AccountNumber: in.AccountNumber,
		Status:        in.Status,
		IsHVT:         in.IsHVT,
		Feature:       feature,
	}
	return
}

type UpdateAccountIn struct {
	AccountNumber string        `json:"-" params:"accountNumber" example:"21100100000001" validate:"required"`
	IsHVT         *bool         `json:"isHvt" example:"true" validate:"required"`
	Status        string        `json:"status" example:"active"`
	Feature       WalletFeature `json:"features"`
}

type UpdateAccountBySubCategoryRequest struct {
	Code            string  `json:"-" params:"subCategoryCode" validate:"required" example:"10000"`
	ProductTypeName *string `json:"productTypeName" validate:"omitempty" example:"CIH Lender"`
	Currency        *string `json:"currency" validate:"omitempty" example:"IDR"`
}

func (in *UpdateAccountBySubCategoryRequest) TransformAndValidate() (out UpdateAccountBySubCategoryIn) {
	return UpdateAccountBySubCategoryIn{
		Code:            in.Code,
		ProductTypeName: in.ProductTypeName,
		Currency:        in.Currency,
	}
}

type UpdateAccountBySubCategoryIn struct {
	Code            string  `json:"code" validate:"required" example:"10000"`
	ProductTypeName *string `json:"productTypeName" example:"1001"`
	Currency        *string `json:"currency" example:"IDR"`
}
