package models

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"encoding/base64"
	"fmt"
)

// ToFilterOpts converts GetMoneyFlowSummaryRequest to MoneyFlowSummaryFilterOptions
func (req GetMoneyFlowSummaryRequest) ToFilterOpts() (*MoneyFlowSummaryFilterOptions, error) {
	opts := &MoneyFlowSummaryFilterOptions{
		PaymentType: req.PaymentType,
		Status:      req.Status,
		Limit:       req.Limit,
	}

	// Validate and parse transactionSourceCreationDate
	if req.TransactionSourceCreationDate != "" {
		date, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.TransactionSourceCreationDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.TransactionSourceCreationDate))
		}
		opts.TransactionSourceCreationDate = &date
	}

	// Set default limit
	if opts.Limit == 0 {
		opts.Limit = 10
	}

	if opts.Limit < 0 {
		return nil, GetErrMap(ErrKeyLimitMustBeGreaterThanZero)
	}

	// Use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// Forward pagination
	if req.NextCursor != "" {
		cursor, err := decodeMoneFlowSummaryCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}
		opts.Cursor = cursor
	}

	// Backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		cursor, err := decodeMoneFlowSummaryCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}
		cursor.IsBackward = true
		opts.Cursor = cursor
	}

	return opts, nil
}

// decodeMoneFlowSummaryCursor decodes base64 encoded cursor
func decodeMoneFlowSummaryCursor(cursor string) (*MoneFlowSummaryCursor, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cursor string: %w", err)
	}

	id := string(decodedBytes)
	if id == "" {
		return nil, fmt.Errorf("failed to parse cursor string: invalid format")
	}

	return &MoneFlowSummaryCursor{
		ID:         id,
		IsBackward: false,
	}, nil
}

// String encodes cursor to base64 string
func (c MoneFlowSummaryCursor) String() string {
	return base64.StdEncoding.EncodeToString([]byte(c.ID))
}

// GetCursor returns encoded cursor
func (m MoneyFlowSummaryOut) GetCursor() string {
	cursor := MoneFlowSummaryCursor{
		ID:         m.ID,
		IsBackward: false,
	}
	return cursor.String()
}
