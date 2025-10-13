package services

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type MoneyFlowService interface {
	ProcessTransactionNotification(ctx context.Context, notification models.TransactionNotificationPayload) error
	IsEligibleTransactionType(transactionType string) bool
}

type moneyFlowCalc service

var _ MoneyFlowService = (*moneyFlowCalc)(nil)

// Eligible transaction types
var eligibleTransactionTypes = map[string]bool{
	"SIVEA": true,
	"ITDEP": true,
	"ITRTP": true,
}

func (mf *moneyFlowCalc) IsEligibleTransactionType(transactionType string) bool {
	return eligibleTransactionTypes[transactionType]
}

func (mf *moneyFlowCalc) ProcessTransactionNotification(ctx context.Context, notification models.TransactionNotificationPayload) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Validate notification status
	if notification.Status != "SUCCESS" {
		return fmt.Errorf("transaction status is not SUCCESS: %s", notification.Status)
	}

	// Process each transaction
	for _, trx := range notification.AcuanData.Body.Data.Order.Transactions {
		// Check if transaction type is eligible
		if !mf.IsEligibleTransactionType(string(trx.TransactionType)) {
			continue
		}

		var amount float64

		// Parse amount
		amount = trx.Amount.InexactFloat64()
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}

		// Parse transaction time
		if trx.TransactionTime.Time == nil {
			return fmt.Errorf("transaction time is nil")
		}
		transactionTime := *trx.TransactionTime.Time
		if transactionTime.IsZero() {
			return fmt.Errorf("invalid transaction time")
		}

		// Get transaction date (without time)
		transactionDate := time.Date(
			transactionTime.Year(),
			transactionTime.Month(),
			transactionTime.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		// Process in atomic transaction
		err = mf.srv.sqlRepo.Atomic(ctx, func(ctx context.Context, r repositories.SQLRepository) error {
			mfRepo := r.GetMoneyFlowCalcRepository()

			// Check if transaction already processed
			isProcessed, err := mfRepo.IsTransactionProcessed(ctx, trx.Id.String())
			if err != nil {
				return fmt.Errorf("failed to check if transaction is processed: %w", err)
			}

			if isProcessed {
				return nil // Skip if already processed
			}

			// Get or create summary
			summary, err := mfRepo.GetSummaryByTypeAndDate(ctx, string(trx.TransactionType), transactionDate)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get summary: %w", err)
			}

			var summaryID uint64
			if summary == nil {
				// Create new summary
				summaryID, err = mfRepo.CreateSummary(ctx, models.CreateMoneyFlowSummary{
					TransactionType: string(trx.TransactionType),
					TransactionDate: transactionDate,
					TotalTransfer:   amount,
				})
				if err != nil {
					return fmt.Errorf("failed to create summary: %w", err)
				}
			} else {
				// Update existing summary
				summaryID = summary.ID
				err = mfRepo.UpdateSummary(ctx, models.UpdateMoneyFlowSummary{
					ID:              summary.ID,
					TransactionType: string(trx.TransactionType),
					TransactionDate: transactionDate,
					TotalTransfer:   summary.TotalTransfer + amount,
				})
				if err != nil {
					return fmt.Errorf("failed to update summary: %w", err)
				}
			}

			// Create detailed summary
			err = mfRepo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
				SummaryID:       summaryID,
				TransactionID:   trx.Id.String(),
				RefNumber:       notification.Identifier,
				Amount:          amount,
				TransactionTime: transactionTime,
			})
			if err != nil {
				return fmt.Errorf("failed to create detailed summary: %w", err)
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
