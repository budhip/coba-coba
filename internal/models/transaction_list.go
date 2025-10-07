package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/shopspring/decimal"
)

const DefaultThresholdStatusCountTransaction uint = 50_000

type GetTransactionOut struct {
	// DatabaseID is identifier in DB the only purpose is for cursor pagination
	DatabaseID                 uint64
	TransactionID              string
	FromAccount                string
	FromAccountName            string
	FromAccountProductTypeName string
	ToAccount                  string
	ToAccountName              string
	ToAccountProductTypeName   string
	Currency                   string
	Amount                     decimal.Decimal
	RefNumber                  string
	OrderType                  string
	OrderTypeName              string
	TransactionType            string
	TransactionTypeName        string
	Method                     string
	TransactionTime            time.Time
	TransactionDate            time.Time
	Status                     string
	Description                string
	Metadata                   string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetStatusName return status name, currently only support SUCCESS
func (t GetTransactionOut) GetStatusName() string {
	title, ok := MapTransactionStatus[TransactionStatus(t.Status)]
	if !ok {
		return t.Status
	}

	return title
}

type DoGetTransactionResponse struct {
	Kind                       string         `json:"kind" example:"transaction"`
	TransactionID              string         `json:"transactionId" example:"c172ca84-9ae2-489c-ae4f-8ef372a109ae"`
	RefNumber                  string         `json:"refNumber" example:"55aa66bb-e6e0-4065-9f4a-64182e97e9d9"`
	OrderType                  string         `json:"orderType" example:"TOPUP"`
	OrderTypeName              string         `json:"orderTypeName" example:"TOPUP"`
	Method                     string         `json:"method" example:"TOPUP.VA"`
	TransactionType            string         `json:"transactionType" example:"TOPUP"`
	TransactionTypeName        string         `json:"transactionTypeName" example:"TOPUP"`
	TransactionDate            string         `json:"transactionDate" example:"2023-10-25"`
	TransactionTime            string         `json:"transactionTime" example:"2023-10-25 08:08:26"`
	FromAccount                string         `json:"fromAccount" example:"189513"`
	FromAccountName            string         `json:"fromAccountName" example:"John"`
	FromAccountProductTypeName string         `json:"fromAccountProductTypeName" example:"valid"`
	ToAccount                  string         `json:"toAccount" example:"222000000069"`
	ToAccountName              string         `json:"toAccountName" example:"John"`
	ToAccountProductTypeName   string         `json:"toAccountProductTypeName" example:"valid"`
	Currency                   string         `json:"currency" example:"IDR"`
	Amount                     string         `json:"amount" example:"50777"`
	Status                     string         `json:"status" example:"1"`
	Description                string         `json:"description" example:"Topup from VA"`
	Metadata                   map[string]any `json:"metadata" swaggertype:"object,string" example:"key:value"`
	CreatedAt                  string         `json:"createdAt" example:"2006-01-02 15:04:05"`
	UpdatedAt                  string         `json:"updatedAt" example:"2006-01-02 15:04:05"`
}

type DoGetListTransactionRequest struct {
	Search           string   `query:"search" example:"value of accountNumber refNumber transactionId"`
	SearchBy         string   `query:"searchBy" example:"accountNumber refNumber transactionId"`
	OrderType        string   `query:"orderType" example:"TOPUP"`
	TransactionType  string   `query:"transactionType" example:"TOPUP"` // Deprecated: /v1/transaction not use this anymore
	TransactionTypes []string `query:"transactionTypes" example:"TOPUP"`
	StartDate        string   `query:"startDate" example:"2023-01-01"`
	EndDate          string   `query:"endDate" example:"2023-01-07"`
	Limit            int      `query:"limit" example:"10"`
	ProductTypeName  string   `query:"productTypeName" example:"Poket"`
	NextCursor       string   `query:"nextCursor" example:"abc"`
	PrevCursor       string   `query:"prevCursor" example:"cba"`
}

type DoGetStatusCountTransactionRequest struct {
	DoGetListTransactionRequest

	Threshold uint `query:"threshold" example:"10"`
}

func (req DoGetStatusCountTransactionRequest) ToFilterOpts() (*TransactionFilterOptions, uint, error) {
	opts, err := req.DoGetListTransactionRequest.ToFilterOpts()
	if err != nil {
		return nil, 0, err
	}

	threshold := req.Threshold

	if threshold == 0 {
		threshold = DefaultThresholdStatusCountTransaction
	}

	return opts, threshold, nil
}

type StatusCountTransaction struct {
	ExceedThreshold bool
	Threshold       uint
}

func (s StatusCountTransaction) ToResponse() DoGetStatusCountTransactionResponse {
	return DoGetStatusCountTransactionResponse{
		Kind:            "status-count-transaction",
		ExceedThreshold: s.ExceedThreshold,
		Threshold:       s.Threshold,
	}

}

type DoGetStatusCountTransactionResponse struct {
	Kind            string `json:"kind" example:"status-count-transaction"`
	ExceedThreshold bool   `json:"exceedThreshold" example:"false"`
	Threshold       uint   `json:"threshold" example:"500000"`
}

type TransactionFilterOptions struct {
	Search           string
	SearchBy         string
	OrderType        string
	TransactionType  string // Deprecated: /v1/transaction not use this anymore
	TransactionTypes []string
	StartDate        *time.Time
	EndDate          *time.Time

	// TransactionDate is filtering using single date
	TransactionDate *time.Time
	Limit           int
	ProductTypeName string

	// Filter only AMF transaction
	OnlyAMF bool

	Cursor *TransactionCursor
}

type TransactionStreamAllOptions struct {
	TransactionDate time.Time
	TransactionType string
}

type TransactionCursor struct {
	TransactionDate time.Time
	DatabaseID      uint64

	IsBackward bool
}

func (t TransactionCursor) String() string {
	strCursor := fmt.Sprintf(
		"%s|%d",
		t.TransactionDate.Format(common.DateFormatYYYYMMDD),
		t.DatabaseID)
	return base64.StdEncoding.EncodeToString([]byte(strCursor))
}

func decodeTransactionCursor(cursor string) (tc *TransactionCursor, err error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset string: %w", err)
	}

	splitCursor := strings.Split(string(decodedBytes), "|")
	if len(splitCursor) != 2 {
		return nil, fmt.Errorf("failed to parse offset string: invalid format")
	}

	decodedDate, err := time.Parse(common.DateFormatYYYYMMDD, splitCursor[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset date: %w", err)
	}

	id, err := strconv.ParseUint(splitCursor[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset id: %w", err)
	}

	tc = &TransactionCursor{
		TransactionDate: decodedDate,
		DatabaseID:      id,
	}

	return tc, nil
}

func (req DoGetListTransactionRequest) ToFilterOpts() (*TransactionFilterOptions, error) {
	opts := &TransactionFilterOptions{
		Search:           req.Search,
		SearchBy:         req.SearchBy,
		OrderType:        req.OrderType,
		TransactionTypes: req.TransactionTypes,
		Limit:            req.Limit,
		ProductTypeName:  req.ProductTypeName,
	}

	if req.Limit < 0 {
		return nil, GetErrMap(ErrKeyLimitMustBeGreaterThanZero)
	}

	if req.StartDate == "" && req.EndDate != "" || req.StartDate != "" && req.EndDate == "" {
		return nil, GetErrMap(ErrKeyStartDateAndEndDateRequiredIfOneIsFilled)
	}

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.StartDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.StartDate))
		}
		opts.StartDate = &startDate

		endDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.EndDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.EndDate))
		}
		opts.EndDate = &endDate

		if startDate.After(endDate) {
			return nil, GetErrMap(ErrKeyStartDateIsAfterEndDate)
		}

		if common.GetTotalDiffDayBetweenTwoDate(startDate, endDate) > 7 {
			return nil, GetErrMap(ErrKeyDateRangeMax7Days)
		}
	}

	if req.Limit == 0 {
		// default limit
		opts.Limit = 10
	}

	// use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// forward pagination
	if req.NextCursor != "" {
		tc, err := decodeTransactionCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}

		opts.Cursor = tc
	}

	// backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		tc, err := decodeTransactionCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}

		tc.IsBackward = true
		opts.Cursor = tc
	}

	return opts, nil
}

func (req DoGetListTransactionRequest) ToDownloadFilterOpts() (*TransactionFilterOptions, error) {
	opts := &TransactionFilterOptions{
		Search:           req.Search,
		SearchBy:         req.SearchBy,
		OrderType:        req.OrderType,
		TransactionType:  req.TransactionType,
		TransactionTypes: req.TransactionTypes,
		ProductTypeName:  req.ProductTypeName,
		Limit:            0,
	}

	if req.StartDate == "" && req.EndDate != "" || req.StartDate != "" && req.EndDate == "" {
		return nil, GetErrMap(ErrKeyStartDateAndEndDateRequiredIfOneIsFilled)
	} else if req.StartDate == "" && req.EndDate == "" {
		now, _ := common.NowZeroTime()
		startDate := now.AddDate(0, 0, -7)
		opts.StartDate = &startDate
		opts.EndDate = &now
	}

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.StartDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.StartDate))
		}
		opts.StartDate = &startDate

		endDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.EndDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.EndDate))
		}
		opts.EndDate = &endDate

		if startDate.After(endDate) {
			return nil, GetErrMap(ErrKeyStartDateIsAfterEndDate)
		}

		if common.GetTotalDiffDayBetweenTwoDate(startDate, endDate) > 7 {
			return nil, GetErrMap(ErrKeyDateRangeMax7Days)
		}
	}

	return opts, nil
}

func (t GetTransactionOut) GetCursor() string {
	tc := TransactionCursor{
		TransactionDate: t.TransactionDate,
		DatabaseID:      t.DatabaseID,
	}
	return tc.String()
}

func (t GetTransactionOut) ToModelResponse() DoGetTransactionResponse {
	var metadata map[string]interface{}
	if t.Metadata != "" {
		_ = json.Unmarshal([]byte(t.Metadata), &metadata)
	}

	return DoGetTransactionResponse{
		Kind:                       kindTransaction,
		TransactionID:              t.TransactionID,
		RefNumber:                  t.RefNumber,
		OrderType:                  t.OrderType,
		OrderTypeName:              t.OrderTypeName,
		Method:                     t.Method,
		TransactionType:            t.TransactionType,
		TransactionTypeName:        t.TransactionTypeName,
		TransactionDate:            t.TransactionDate.In(common.GetLocation()).Format(common.DateFormatYYYYMMDD),
		TransactionTime:            t.TransactionTime.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTimeAndOffset),
		FromAccount:                t.FromAccount,
		FromAccountName:            t.FromAccountName,
		FromAccountProductTypeName: t.FromAccountProductTypeName,
		ToAccount:                  t.ToAccount,
		ToAccountName:              t.ToAccountName,
		ToAccountProductTypeName:   t.ToAccountProductTypeName,
		Amount:                     t.Amount.String(),
		Currency:                   t.Currency,
		Status:                     t.GetStatusName(),
		Description:                t.Description,
		Metadata:                   metadata,
		CreatedAt:                  t.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTimeAndOffset),
		UpdatedAt:                  t.UpdatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTimeAndOffset),
	}
}

type DownloadTransactionRequest struct {
	Options TransactionFilterOptions
	Writer  io.Writer
}

type TransactionGetByTypeAndRefNumberRequest struct {
	TransactionType string `params:"transactionType" example:"INVESTMENT" validate:"required"`
	RefNumber       string `params:"refNumber" example:"123456" validate:"required"`
}
