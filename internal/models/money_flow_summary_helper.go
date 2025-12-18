package models

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/pagination"
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

		var decodedStr string
		decodedBytes, err := base64.StdEncoding.DecodeString(cursorStr)
		if err != nil {
			decodedBytes, err = base64.URLEncoding.DecodeString(cursorStr)
			if err != nil {
				decodedStr = cursorStr
			} else {
				decodedStr = string(decodedBytes)
			}
		} else {
			decodedStr = string(decodedBytes)
		}

		// Parse format: "created_at|id"
		parts := strings.Split(decodedStr, "|")
		if len(parts) == 2 {
			// New format with ID
			// Try RFC3339Nano first (with microseconds), fallback to RFC3339
			createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
			if err != nil {
				// Fallback for old cursors without microseconds
				createdAt, err = time.Parse(time.RFC3339, parts[0])
				if err != nil {
					return nil, fmt.Errorf("invalid cursor created_at format: %w", err)
				}
			}

			opts.Cursor = &MoneyFlowSummaryCursor{
				CreatedAt:  createdAt,
				ID:         parts[1],
				IsBackward: cursor.IsBackward(),
			}
		} else {
			// Old format (backward compatibility) - only created_at
			// Try RFC3339Nano first, fallback to RFC3339
			createdAt, err := time.Parse(time.RFC3339Nano, decodedStr)
			if err != nil {
				createdAt, err = time.Parse(time.RFC3339, decodedStr)
				if err != nil {
					return nil, fmt.Errorf("invalid cursor format: %w", err)
				}
			}

			opts.Cursor = &MoneyFlowSummaryCursor{
				CreatedAt:  createdAt,
				ID:         "",
				IsBackward: cursor.IsBackward(),
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
