package models

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/pagination"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
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
		cursorStr := cursor.GetID()

		// Check if cursor is already decoded (plain text format: "date:id")
		if strings.Contains(cursorStr, ":") && strings.Contains(cursorStr, "T") {
			// Already decoded, parse directly
			parts := strings.SplitN(cursorStr, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cursor format: expected date:id")
			}

			// Parse the date
			cursorDate, err := time.Parse(time.RFC3339, parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid cursor date format: %w", err)
			}

			opts.Cursor = &MoneyFlowSummaryCursor{
				TransactionSourceCreationDate: cursorDate,
				ID:                            parts[1],
				IsBackward:                    cursor.IsBackward(),
			}
		} else {
			// Try to decode base64
			decodedBytes, err := base64.RawURLEncoding.DecodeString(cursorStr)
			if err != nil {
				// Try with padding
				decodedBytes, err = base64.URLEncoding.DecodeString(cursorStr)
				if err != nil {
					return nil, fmt.Errorf("invalid cursor format: %w", err)
				}
			}

			decodedStr := string(decodedBytes)
			parts := strings.SplitN(decodedStr, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cursor format: expected date:id")
			}

			// Parse the date
			cursorDate, err := time.Parse(time.RFC3339, parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid cursor date format: %w", err)
			}

			opts.Cursor = &MoneyFlowSummaryCursor{
				TransactionSourceCreationDate: cursorDate,
				ID:                            parts[1],
				IsBackward:                    cursor.IsBackward(),
			}
		}
	}

	return opts, nil
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
