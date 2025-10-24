package models

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/pagination"
	"encoding/base64"
	"fmt"
)

// ToFilterOpts converts GetMoneyFlowSummaryRequest to MoneyFlowSummaryFilterOptions
func (req GetMoneyFlowSummaryRequest) ToFilterOpts() (*MoneyFlowSummaryFilterOptions, error) {
	opts := &MoneyFlowSummaryFilterOptions{
		PaymentType: req.PaymentType,
		Status:      req.Status,
	}

	// Parse start date
	if req.TransactionSourceCreationDateStart != "" {
		date, err := common.ParseStringToDatetime(constants.DateFormatYYYYMMDD, req.TransactionSourceCreationDateStart)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("transactionSourceCreationDateStart %s format must be YYYY-MM-DD", req.TransactionSourceCreationDateStart))
		}
		opts.TransactionSourceCreationDateStart = &date
	}

	// Parse end date
	if req.TransactionSourceCreationDateEnd != "" {
		date, err := common.ParseStringToDatetime(constants.DateFormatYYYYMMDD, req.TransactionSourceCreationDateEnd)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("transactionSourceCreationDateEnd %s format must be YYYY-MM-DD", req.TransactionSourceCreationDateEnd))
		}
		opts.TransactionSourceCreationDateEnd = &date
	}

	paginationOpts := pagination.Options{
		Limit:      req.Limit,
		NextCursor: req.NextCursor,
		PrevCursor: req.PrevCursor,
	}

	cursor, limit, err := paginationOpts.BuildCursorAndLimit()
	if err != nil {
		return nil, err
	}

	opts.Limit = limit

	if cursor != nil {
		opts.Cursor = &MoneyFlowSummaryCursor{
			ID:         cursor.GetID(),
			IsBackward: cursor.IsBackward(),
		}
	}

	return opts, nil
}

// GetCursor returns encoded cursor
func (m MoneyFlowSummaryOut) GetCursor() string {
	return base64.StdEncoding.EncodeToString([]byte(m.ID))
}

// ToFilterOpts converts request to filter options
func (req DoGetDetailedTransactionsBySummaryIDRequest) ToFilterOpts() (*DetailedTransactionFilterOptions, error) {
	opts := &DetailedTransactionFilterOptions{
		SummaryID: req.SummaryID,
		RefNumber: req.RefNumber,
	}

	paginationOpts := pagination.Options{
		Limit:      req.Limit,
		NextCursor: req.NextCursor,
		PrevCursor: req.PrevCursor,
	}

	cursor, limit, err := paginationOpts.BuildCursorAndLimit()
	if err != nil {
		return nil, err
	}

	opts.Limit = limit

	if cursor != nil {
		opts.Cursor = &DetailedTransactionCursor{
			ID:         cursor.GetID(),
			IsBackward: cursor.IsBackward(),
		}
	}

	return opts, nil
}
