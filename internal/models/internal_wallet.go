package models

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

type ListWalletTrxByAccountNumberRequest struct {
	AccountNumber   string `params:"accountNumber" validate:"required" example:"21100100000001"`
	Limit           int    `query:"limit" example:"10"`
	Status          string `query:"status" validate:"omitempty,oneof=CANCEL PENDING SUCCESS" example:"PENDING"`
	StartDate       string `query:"startDate" example:"2023-01-01"`
	EndDate         string `query:"endDate" example:"2023-01-07"`
	TransactionType string `query:"transactionType" example:"TUPVA"`
	SortBy          string
	SortDirection   string
	NextCursor      string `query:"nextCursor" example:"abc"`
	PrevCursor      string `query:"prevCursor" example:"cba"`
}

type ListWalletTrxRequest struct {
	AccountNumbers   string `params:"accountNumbers" example:"21100100000001,21100100000002,21100100000003"`
	TransactionTypes string `params:"transactionTypes" example:"VOLTR,TUPVA"`
	Limit            int    `query:"limit" example:"10"`
	Status           string `query:"status" validate:"omitempty,oneof=CANCEL PENDING SUCCESS" example:"PENDING"`
	StartDate        string `query:"startDate" example:"2023-01-01"`
	EndDate          string `query:"endDate" example:"2023-01-07"`
	TransactionType  string `query:"transactionType" example:"TUPVA"`
	SortBy           string
	SortDirection    string
	NextCursor       string `query:"nextCursor" example:"abc"`
	PrevCursor       string `query:"prevCursor" example:"cba"`
}

type WalletTrxCursor struct {
	TransactionTime time.Time
	Id              string

	IsBackward bool
}

func (w WalletTrxCursor) String() string {
	strCursor := fmt.Sprintf(
		"%s|%s",
		w.TransactionTime.Format(time.RFC3339Nano),
		w.Id)
	return base64.StdEncoding.EncodeToString([]byte(strCursor))
}

func decodeWalletTrxCursor(cursor string) (*WalletTrxCursor, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset string: %w", err)
	}

	splitCursor := strings.Split(string(decodedBytes), "|")
	if len(splitCursor) != 2 {
		return nil, fmt.Errorf("failed to parse offset string: invalid format")
	}

	decodedTime, err := time.Parse(time.RFC3339Nano, splitCursor[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse offset date: %w", err)
	}

	return &WalletTrxCursor{
		TransactionTime: decodedTime,
		Id:              splitCursor[1],
	}, nil
}

func (req ListWalletTrxByAccountNumberRequest) ToFilterOpts() (*WalletTrxFilterOptions, error) {
	opts := &WalletTrxFilterOptions{
		AccountNumber:   req.AccountNumber,
		Limit:           req.Limit,
		Status:          req.Status,
		TransactionType: req.TransactionType,
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

		endDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDDWithTime, fmt.Sprintf("%s 23:59:59", req.EndDate))
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.EndDate))
		}
		opts.EndDate = &endDate

		if startDate.After(endDate) {
			return nil, GetErrMap(ErrKeyStartDateIsAfterEndDate)
		}
	}

	if req.Limit < 0 {
		return nil, GetErrMap(ErrKeyLimitMustBeGreaterThanZero)
	}

	// default limit
	if req.Limit == 0 {
		opts.Limit = 10
	}

	// default sortBy
	if req.SortBy == "" {
		opts.SortBy = "createdDate"
	}

	// default sort direction
	if req.SortDirection == "" {
		opts.SortDirection = SortByDESC
	}

	// use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// forward pagination
	if req.NextCursor != "" {
		wtc, err := decodeWalletTrxCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}

		opts.Cursor = wtc
	}

	// backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		wtc, err := decodeWalletTrxCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}

		wtc.IsBackward = true
		opts.Cursor = wtc
	}

	return opts, nil
}

func (req ListWalletTrxRequest) ToFilterOpts() (*WalletTrxFilterOptions, error) {
	opts := &WalletTrxFilterOptions{
		Limit:           req.Limit,
		Status:          req.Status,
		TransactionType: req.TransactionType,
	}

	if req.AccountNumbers != "" {
		accountNumbers := strings.TrimSpace(req.AccountNumbers)
		opts.AccountNumbers = strings.Split(accountNumbers, ",")
	}

	if req.TransactionTypes != "" {
		transactionTypes := strings.TrimSpace(req.TransactionTypes)
		opts.TransactionTypes = strings.Split(transactionTypes, ",")
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

		endDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDDWithTime, fmt.Sprintf("%s 23:59:59", req.EndDate))
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.EndDate))
		}
		opts.EndDate = &endDate

		if startDate.After(endDate) {
			return nil, GetErrMap(ErrKeyStartDateIsAfterEndDate)
		}
	}

	if req.Limit < 0 {
		return nil, GetErrMap(ErrKeyLimitMustBeGreaterThanZero)
	}

	// default limit
	if req.Limit == 0 {
		opts.Limit = 10
	}

	// default sortBy
	if req.SortBy == "" {
		opts.SortBy = "createdDate"
	}

	// default sort direction
	if req.SortDirection == "" {
		opts.SortDirection = SortByDESC
	}

	// use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// forward pagination
	if req.NextCursor != "" {
		wtc, err := decodeWalletTrxCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}

		opts.Cursor = wtc
	}

	// backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		wtc, err := decodeWalletTrxCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}

		wtc.IsBackward = true
		opts.Cursor = wtc
	}

	return opts, nil
}

type WalletTrxFilterOptions struct {
	AccountNumber    string
	Limit            int
	SortBy           string
	SortDirection    string
	Status           string
	TransactionType  string
	StartDate        *time.Time
	EndDate          *time.Time
	AccountNumbers   []string
	TransactionTypes []string

	Cursor *WalletTrxCursor
}

type ListWalletTrxByAccountNumberResponse struct {
	Kind                string                  `json:"kind"`
	TransactionDate     string                  `json:"transactionDate"`
	TransactionTime     string                  `json:"transactionTime"`
	TransactionType     string                  `json:"transactionType"`
	Amount              Amount                  `json:"amount"`
	Amounts             Amounts                 `json:"amounts"`
	Status              WalletTransactionStatus `json:"status"`
	TransactionWalletId string                  `json:"transactionWalletId"`
	RefNumber           string                  `json:"refNumber"`
	Description         string                  `json:"description"`
	TransactionFlow     TransactionFlow         `json:"transactionFlow"`
	Metadata            WalletMetadata          `json:"metadata"`
}
