package services

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/shopspring/decimal"
)

// TransactionProcessor handles transaction processing logic
type TransactionProcessor struct {
	repo repositories.MoneyFlowRepository
}

// NewTransactionProcessor creates a new TransactionProcessor instance
func NewTransactionProcessor(repo repositories.MoneyFlowRepository) *TransactionProcessor {
	return &TransactionProcessor{repo: repo}
}

// ProcessOrUpdate processes new transaction or updates existing one with failed/rejected handling
func (tp *TransactionProcessor) ProcessOrUpdate(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {

	currentDate := summaryData.TransactionSourceCreationDate.Truncate(24 * time.Hour)

	// STEP 1: Check if transaction already processed for current date
	processed, err := tp.repo.GetTransactionProcessed(
		ctx,
		summaryData.TransactionType,
		summaryData.TransactionSourceCreationDate,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check transaction: %w", err)
	}

	// STEP 2: If transaction exists for current date, just update it
	if processed != nil {
		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found existing transaction for current date",
			xlog.String("summary_id", processed.ID),
			xlog.Time("transaction_date", currentDate))

		// Update existing summary by adding the amount
		totalAmount := amount.Add(processed.TotalTransfer)
		updateReq := models.MoneyFlowSummaryUpdate{
			TotalTransfer: &totalAmount,
		}

		err = tp.repo.UpdateSummary(ctx, processed.ID, updateReq)
		if err != nil {
			return "", fmt.Errorf("failed to update summary: %w", err)
		}

		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Updated existing summary",
			xlog.String("summary_id", processed.ID),
			xlog.String("previous_total", processed.TotalTransfer.String()),
			xlog.String("new_total", totalAmount.String()))

		// Create detailed summary for this transaction
		err = tp.repo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
			SummaryID:          processed.ID,
			AcuanTransactionID: acuanTransactionID,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create detailed summary: %w", err)
		}

		return processed.ID, nil
	}

	// STEP 3: No transaction for current date, check for IN_PROGRESS transactions first
	hasInProgress, err := tp.repo.HasInProgressTransaction(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check in-progress transaction: %w", err)
	}

	if hasInProgress {
		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found IN_PROGRESS transaction, not setting relation",
			xlog.String("transaction_type", summaryData.TransactionType),
			xlog.String("payment_type", summaryData.PaymentType))

		// Create without relation ID because there's an IN_PROGRESS transaction
		summaryID, err := tp.createNewSummary(ctx, summaryData, acuanTransactionID, amount)
		return summaryID, err
	}

	// STEP 4: Check for FAILED/REJECTED from previous dates
	var relatedFailedOrRejectedID *string

	failedOrRejected, err := tp.repo.GetLastFailedOrRejectedTransaction(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check failed/rejected transaction: %w", err)
	}

	// STEP 5: If found FAILED/REJECTED, check if we should set relation
	if failedOrRejected != nil {
		failedDate := failedOrRejected.TransactionSourceCreationDate.Truncate(24 * time.Hour)

		// Only set relation if FAILED/REJECTED is from a DIFFERENT (previous) date
		if failedDate.Before(currentDate) {
			// Check if there's already a PENDING transaction created after this specific FAILED/REJECTED
			hasPendingAfter, err := tp.repo.HasPendingTransactionAfterFailedOrRejected(
				ctx,
				summaryData.TransactionType,
				summaryData.PaymentType,
				failedOrRejected.ID,
			)
			if err != nil {
				return "", fmt.Errorf("failed to check pending after failed/rejected: %w", err)
			}

			// Only set relation if there's NO PENDING yet after this specific FAILED/REJECTED
			if !hasPendingAfter {
				relatedFailedOrRejectedID = &failedOrRejected.ID
				summaryData.RelatedFailedOrRejectedSummaryID = relatedFailedOrRejectedID

				xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Setting relation to failed/rejected (first transaction after this failure)",
					xlog.String("failed_rejected_id", failedOrRejected.ID),
					xlog.String("status", failedOrRejected.MoneyFlowStatus),
					xlog.Time("failed_date", failedDate),
					xlog.Time("current_date", currentDate),
					xlog.String("transaction_type", failedOrRejected.TransactionType),
					xlog.String("payment_type", failedOrRejected.PaymentType))
			} else {
				xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found failed/rejected but PENDING already exists after it, not setting relation",
					xlog.String("failed_rejected_id", failedOrRejected.ID),
					xlog.Time("failed_date", failedDate),
					xlog.Time("current_date", currentDate))
			}
		} else {
			xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found failed/rejected but same date, not setting relation",
				xlog.String("failed_rejected_id", failedOrRejected.ID),
				xlog.Time("failed_date", failedDate),
				xlog.Time("current_date", currentDate))
		}
	}

	// STEP 6: Create new summary
	summaryID, err := tp.createNewSummary(ctx, summaryData, acuanTransactionID, amount)
	if err != nil {
		return "", err
	}

	xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Created new summary",
		xlog.String("summary_id", summaryID),
		xlog.Time("transaction_date", currentDate),
		xlog.String("transaction_type", summaryData.TransactionType),
		xlog.String("payment_type", summaryData.PaymentType),
		xlog.String("amount", amount.String()),
		xlog.String("related_failed_rejected_id", func() string {
			if relatedFailedOrRejectedID != nil {
				return *relatedFailedOrRejectedID
			}
			return "none"
		}()))

	return summaryID, nil
}

// createNewSummary is a helper method to create new summary and detailed summary
func (tp *TransactionProcessor) createNewSummary(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {
	// Create new summary
	summaryID, err := tp.repo.CreateSummary(ctx, summaryData)
	if err != nil {
		return "", fmt.Errorf("failed to create summary: %w", err)
	}

	// Create detailed summary
	err = tp.repo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
		SummaryID:          summaryID,
		AcuanTransactionID: acuanTransactionID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create detailed summary: %w", err)
	}

	return summaryID, nil
}
