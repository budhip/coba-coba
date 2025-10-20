package services

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

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

// ProcessOrUpdate processes new transaction or updates existing one
func (tp *TransactionProcessor) ProcessOrUpdate(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {
	// Check if transaction already processed
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
