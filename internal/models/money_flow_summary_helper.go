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

	// Validate and parse transactionSourceCreationDate
	if req.TransactionSourceCreationDate != "" {
		date, err := common.ParseStringToDatetime(constants.DateFormatYYYYMMDD, req.TransactionSourceCreationDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.TransactionSourceCreationDate))
		}
		opts.TransactionSourceCreationDate = &date
	}

	// Build pagination using helper
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
	}

	// Build pagination using helper
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
