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

	// STEP 1: Check if there's a last FAILED or REJECTED transaction with same type and payment
	failedOrRejected, err := tp.repo.GetLastFailedOrRejectedTransaction(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check failed/rejected transaction: %w", err)
	}

	var relatedFailedOrRejectedID *string

	if failedOrRejected != nil {
		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found failed/rejected transaction",
			xlog.String("failed_rejected_id", failedOrRejected.ID),
			xlog.String("status", failedOrRejected.MoneyFlowStatus),
			xlog.String("transaction_type", failedOrRejected.TransactionType),
			xlog.String("payment_type", failedOrRejected.PaymentType))

		// STEP 2: Check if there's a PENDING transaction after the FAILED/REJECTED one
		pendingAfterFailed, err := tp.repo.GetPendingTransactionAfterFailed(
			ctx,
			summaryData.TransactionType,
			summaryData.PaymentType,
			failedOrRejected.CreatedAt,
		)
		if err != nil {
			return "", fmt.Errorf("failed to check pending transaction after failed: %w", err)
		}

		if pendingAfterFailed != nil {
			// STEP 3a: Found PENDING after FAILED/REJECTED
			xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Found pending transaction after failed/rejected",
				xlog.String("pending_id", pendingAfterFailed.ID),
				xlog.Time("pending_date", pendingAfterFailed.TransactionSourceCreationDate))

			// Check if the pending transaction date is today
			today := time.Now().Truncate(24 * time.Hour)
			pendingDate := pendingAfterFailed.TransactionSourceCreationDate.Truncate(24 * time.Hour)

			if pendingDate.Equal(today) || pendingDate.Equal(summaryData.TransactionSourceCreationDate.Truncate(24*time.Hour)) {
				// Update existing PENDING transaction by adding the amount
				totalAmount := amount.Add(pendingAfterFailed.TotalTransfer)
				updateReq := models.MoneyFlowSummaryUpdate{
					TotalTransfer: &totalAmount,
				}

				err = tp.repo.UpdateSummary(ctx, pendingAfterFailed.ID, updateReq)
				if err != nil {
					return "", fmt.Errorf("failed to update pending summary: %w", err)
				}

				xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Updated pending transaction amount",
					xlog.String("pending_id", pendingAfterFailed.ID),
					xlog.String("new_total", totalAmount.String()))

				// Create detailed summary for this transaction
				err = tp.repo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
					SummaryID:          pendingAfterFailed.ID,
					AcuanTransactionID: acuanTransactionID,
				})
				if err != nil {
					return "", fmt.Errorf("failed to create detailed summary: %w", err)
				}

				return pendingAfterFailed.ID, nil
			}

			// If pending date is not today, fall through to create new with relation
			xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Pending date is not today, creating new summary with relation")
		}

		// STEP 3b: No PENDING found after FAILED/REJECTED, create new with relation
		relatedFailedOrRejectedID = &failedOrRejected.ID
		summaryData.RelatedFailedOrRejectedSummaryID = relatedFailedOrRejectedID

		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Creating new summary with relation to failed/rejected",
			xlog.String("related_failed_rejected_id", *relatedFailedOrRejectedID))
	}

	// STEP 4: Check if transaction already processed for today (existing logic)
	processed, err := tp.repo.GetTransactionProcessed(
		ctx,
		summaryData.TransactionType,
		summaryData.TransactionSourceCreationDate,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check transaction: %w", err)
	}

	var summaryID string

	if processed == nil {
		// Create new summary
		summaryID, err = tp.repo.CreateSummary(ctx, summaryData)
		if err != nil {
			return "", fmt.Errorf("failed to create summary: %w", err)
		}

		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Created new summary",
			xlog.String("summary_id", summaryID),
			xlog.String("transaction_type", summaryData.TransactionType),
			xlog.String("payment_type", summaryData.PaymentType),
			xlog.String("related_failed_rejected_id", func() string {
				if relatedFailedOrRejectedID != nil {
					return *relatedFailedOrRejectedID
				}
				return "none"
			}()))
	} else {
		// Update existing summary
		totalAmount := amount.Add(processed.TotalTransfer)
		updateReq := models.MoneyFlowSummaryUpdate{
			TotalTransfer: &totalAmount,
		}

		err = tp.repo.UpdateSummary(ctx, processed.ID, updateReq)
		if err != nil {
			return "", fmt.Errorf("failed to update summary: %w", err)
		}
		summaryID = processed.ID

		xlog.Info(ctx, "[MONEY-FLOW-PROCESSOR] Updated existing summary",
			xlog.String("summary_id", summaryID),
			xlog.String("new_total", totalAmount.String()))
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
