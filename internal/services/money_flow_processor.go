package services

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/shopspring/decimal"
)

// TransactionProcessor handles transaction processing logic with clear separation of concerns
type TransactionProcessor struct {
	repo                  repositories.MoneyFlowRepository
	relationshipManager   *RelationshipManager
	transactionRepository *TransactionRepository
}

// NewTransactionProcessor creates a new TransactionProcessor instance
func NewTransactionProcessor(repo repositories.MoneyFlowRepository) *TransactionProcessor {
	return &TransactionProcessor{
		repo:                  repo,
		relationshipManager:   NewRelationshipManager(repo),
		transactionRepository: NewTransactionRepository(repo),
	}
}

// ProcessOrUpdate processes new transaction or updates existing one
func (tp *TransactionProcessor) ProcessOrUpdate(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {
	currentDate := summaryData.TransactionSourceCreationDate.Truncate(24 * time.Hour)

	// Check if transaction already exists for current date
	if existingSummaryID, err := tp.updateExistingIfPresent(ctx, summaryData, acuanTransactionID, amount, currentDate); err != nil {
		return "", err
	} else if existingSummaryID != "" {
		return existingSummaryID, nil
	}

	// Create new transaction with appropriate relationships
	return tp.createNewTransaction(ctx, summaryData, acuanTransactionID, amount, currentDate)
}

// updateExistingIfPresent updates existing transaction if it exists for current date
func (tp *TransactionProcessor) updateExistingIfPresent(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
	currentDate time.Time,
) (string, error) {
	processed, err := tp.repo.GetTransactionProcessed(
		ctx,
		summaryData.TransactionType,
		summaryData.TransactionSourceCreationDate,
	)
	if err != nil {
		return "", fmt.Errorf("failed to check transaction: %w", err)
	}

	if processed == nil {
		return "", nil // No existing transaction
	}

	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Found existing transaction for current date",
		xlog.String("summary_id", processed.ID),
		xlog.Time("transaction_date", currentDate))

	return tp.updateExistingSummary(ctx, processed, acuanTransactionID, amount)
}

// updateExistingSummary updates an existing summary with new amount
func (tp *TransactionProcessor) updateExistingSummary(
	ctx context.Context,
	processed *models.MoneyFlowTransactionProcessed,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {
	// Update total amount
	totalAmount := amount.Add(processed.TotalTransfer)
	updateReq := models.MoneyFlowSummaryUpdate{
		TotalTransfer: &totalAmount,
	}

	if err := tp.repo.UpdateSummary(ctx, processed.ID, updateReq); err != nil {
		return "", fmt.Errorf("failed to update summary: %w", err)
	}

	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Updated existing summary",
		xlog.String("summary_id", processed.ID),
		xlog.String("previous_total", processed.TotalTransfer.String()),
		xlog.String("new_total", totalAmount.String()))

	// Create detailed summary
	if err := tp.transactionRepository.CreateDetailedSummary(ctx, processed.ID, acuanTransactionID); err != nil {
		return "", err
	}

	return processed.ID, nil
}

// createNewTransaction creates a new transaction with relationship checking
func (tp *TransactionProcessor) createNewTransaction(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
	currentDate time.Time,
) (string, error) {
	// Determine relationship to failed/rejected transactions
	relatedID, err := tp.relationshipManager.DetermineRelationship(ctx, summaryData, currentDate)
	if err != nil {
		return "", err
	}

	summaryData.RelatedFailedOrRejectedSummaryID = relatedID

	// Create new summary
	summaryID, err := tp.transactionRepository.CreateNewSummary(ctx, summaryData, acuanTransactionID, amount)
	if err != nil {
		return "", err
	}

	tp.logNewSummaryCreation(ctx, summaryID, summaryData, amount, relatedID, currentDate)

	return summaryID, nil
}

// logNewSummaryCreation logs the creation of a new summary
func (tp *TransactionProcessor) logNewSummaryCreation(
	ctx context.Context,
	summaryID string,
	summaryData models.CreateMoneyFlowSummary,
	amount decimal.Decimal,
	relatedID *string,
	currentDate time.Time,
) {
	relatedIDStr := "none"
	if relatedID != nil {
		relatedIDStr = *relatedID
	}

	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Created new summary",
		xlog.String("summary_id", summaryID),
		xlog.Time("transaction_date", currentDate),
		xlog.String("transaction_type", summaryData.TransactionType),
		xlog.String("payment_type", summaryData.PaymentType),
		xlog.String("amount", amount.String()),
		xlog.String("related_failed_rejected_id", relatedIDStr))
}

// RelationshipManager manages relationships between transactions
type RelationshipManager struct {
	repo repositories.MoneyFlowRepository
}

// NewRelationshipManager creates a new relationship manager
func NewRelationshipManager(repo repositories.MoneyFlowRepository) *RelationshipManager {
	return &RelationshipManager{repo: repo}
}

// DetermineRelationship determines if there should be a relationship to a failed/rejected transaction
func (rm *RelationshipManager) DetermineRelationship(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	currentDate time.Time,
) (*string, error) {
	// Check for IN_PROGRESS transactions first
	if hasInProgress, err := rm.checkInProgressTransaction(ctx, summaryData); err != nil {
		return nil, err
	} else if hasInProgress {
		return nil, nil // No relationship if IN_PROGRESS exists
	}

	// Check for FAILED/REJECTED transactions
	return rm.checkFailedOrRejectedRelationship(ctx, summaryData, currentDate)
}

// checkInProgressTransaction checks for existing IN_PROGRESS transactions
func (rm *RelationshipManager) checkInProgressTransaction(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
) (bool, error) {
	hasInProgress, err := rm.repo.HasInProgressTransaction(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check in-progress transaction: %w", err)
	}

	if hasInProgress {
		xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Found IN_PROGRESS transaction, not setting relation",
			xlog.String("transaction_type", summaryData.TransactionType),
			xlog.String("payment_type", summaryData.PaymentType))
	}

	return hasInProgress, nil
}

// checkFailedOrRejectedRelationship checks for and establishes relationship with failed/rejected transactions
func (rm *RelationshipManager) checkFailedOrRejectedRelationship(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	currentDate time.Time,
) (*string, error) {
	failedOrRejected, err := rm.repo.GetLastFailedOrRejectedTransaction(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check failed/rejected transaction: %w", err)
	}

	if failedOrRejected == nil {
		return nil, nil // No failed/rejected transaction found
	}

	return rm.evaluateRelationship(ctx, failedOrRejected, summaryData, currentDate)
}

// evaluateRelationship evaluates if relationship should be established
func (rm *RelationshipManager) evaluateRelationship(
	ctx context.Context,
	failedOrRejected *models.FailedOrRejectedTransaction,
	summaryData models.CreateMoneyFlowSummary,
	currentDate time.Time,
) (*string, error) {
	failedDate := failedOrRejected.TransactionSourceCreationDate.Truncate(24 * time.Hour)

	// Only set relation if failed/rejected is from a different (previous) date
	if !failedDate.Before(currentDate) {
		rm.logSameDateScenario(ctx, failedOrRejected.ID, failedDate, currentDate)
		return nil, nil
	}

	// Check if there's already a PENDING transaction after this specific failure
	hasPendingAfter, err := rm.repo.HasPendingTransactionAfterFailedOrRejected(
		ctx,
		summaryData.TransactionType,
		summaryData.PaymentType,
		failedOrRejected.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending after failed/rejected: %w", err)
	}

	if hasPendingAfter {
		rm.logPendingExistsScenario(ctx, failedOrRejected.ID, failedDate, currentDate)
		return nil, nil
	}

	// Set the relationship - this is the first transaction after this specific failure
	rm.logRelationshipEstablished(ctx, failedOrRejected, failedDate, currentDate)
	return &failedOrRejected.ID, nil
}

// Logging methods for better observability
func (rm *RelationshipManager) logSameDateScenario(ctx context.Context, failedID string, failedDate, currentDate time.Time) {
	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Found failed/rejected but same date, not setting relation",
		xlog.String("failed_rejected_id", failedID),
		xlog.Time("failed_date", failedDate),
		xlog.Time("current_date", currentDate))
}

func (rm *RelationshipManager) logPendingExistsScenario(ctx context.Context, failedID string, failedDate, currentDate time.Time) {
	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Found failed/rejected but PENDING already exists after it, not setting relation",
		xlog.String("failed_rejected_id", failedID),
		xlog.Time("failed_date", failedDate),
		xlog.Time("current_date", currentDate))
}

func (rm *RelationshipManager) logRelationshipEstablished(
	ctx context.Context,
	failedOrRejected *models.FailedOrRejectedTransaction,
	failedDate, currentDate time.Time,
) {
	xlog.Info(ctx, constants.LogPrefixMoneyFlowProcessor+" Setting relation to failed/rejected (first transaction after this failure)",
		xlog.String("failed_rejected_id", failedOrRejected.ID),
		xlog.String("status", failedOrRejected.MoneyFlowStatus),
		xlog.Time("failed_date", failedDate),
		xlog.Time("current_date", currentDate),
		xlog.String("transaction_type", failedOrRejected.TransactionType),
		xlog.String("payment_type", failedOrRejected.PaymentType))
}

// TransactionRepository handles transaction persistence
type TransactionRepository struct {
	repo repositories.MoneyFlowRepository
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(repo repositories.MoneyFlowRepository) *TransactionRepository {
	return &TransactionRepository{repo: repo}
}

// CreateNewSummary creates a new summary with detailed summary
func (tr *TransactionRepository) CreateNewSummary(
	ctx context.Context,
	summaryData models.CreateMoneyFlowSummary,
	acuanTransactionID string,
	amount decimal.Decimal,
) (string, error) {
	// Create summary
	summaryID, err := tr.repo.CreateSummary(ctx, summaryData)
	if err != nil {
		return "", fmt.Errorf("failed to create summary: %w", err)
	}

	// Create detailed summary
	if err := tr.CreateDetailedSummary(ctx, summaryID, acuanTransactionID); err != nil {
		return "", err
	}

	return summaryID, nil
}

// CreateDetailedSummary creates a detailed summary entry
func (tr *TransactionRepository) CreateDetailedSummary(
	ctx context.Context,
	summaryID string,
	acuanTransactionID string,
) error {
	err := tr.repo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
		SummaryID:          summaryID,
		AcuanTransactionID: acuanTransactionID,
	})
	if err != nil {
		return fmt.Errorf("failed to create detailed summary: %w", err)
	}
	return nil
}
