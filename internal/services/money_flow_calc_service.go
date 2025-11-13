package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	gopaymentlib "bitbucket.org/Amartha/go-payment-lib/payment-api/models/event"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	"github.com/google/uuid"
)

type MoneyFlowService interface {
	ProcessTransactionNotification(ctx context.Context, notification goacuanlib.Payload[goacuanlib.DataOrder]) error
	CheckEligibleTransaction(ctx context.Context, paymentType, breakdownTransactionType string) (*models.BusinessRuleConfig, string, error)
	ProcessTransactionStream(ctx context.Context, event gopaymentlib.Event) error
	GetSummariesList(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) ([]models.MoneyFlowSummaryOut, int, error)
	GetSummaryDetailBySummaryID(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error)
	GetDetailedTransactionsBySummaryID(ctx context.Context, summaryID string, opts models.DetailedTransactionFilterOptions) ([]models.DetailedTransactionOut, int, error)
	UpdateSummary(ctx context.Context, summaryID string, req models.UpdateMoneyFlowSummaryRequest) error
	DownloadDetailedTransactionsBySummaryID(ctx context.Context, req models.DownloadDetailedTransactionsRequest) error
}

type moneyFlowCalc service

var _ MoneyFlowService = (*moneyFlowCalc)(nil)

func (mf *moneyFlowCalc) CheckEligibleTransaction(ctx context.Context, paymentType, breakdownTransactionType string) (*models.BusinessRuleConfig, string, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	businessRulesData, err := mf.loadBusinessRules(ctx)
	if err != nil {
		return nil, "", err
	}

	helper := NewBusinessRulesHelper(businessRulesData)

	if paymentType == "" {
		return helper.GetByTransactionType(breakdownTransactionType)
	}

	config, err := helper.GetByPaymentType(paymentType)
	if err != nil {
		return nil, "", err
	}

	return config, paymentType, nil
}

func (mf *moneyFlowCalc) ProcessTransactionNotification(ctx context.Context, notification goacuanlib.Payload[goacuanlib.DataOrder]) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Validate notification status
	if notification.Body.Data.Order.Transactions[0].Status != 1 {
		xlog.Info(ctx, "[MONEY-FLOW-CALC] Skipping non-success transaction",
			xlog.String("status", string(notification.Body.Data.Order.Transactions[0].Status)),
			xlog.String("ref_number", notification.Body.Data.Order.RefNumber))
		return nil
	}

	// Validate order time
	if notification.Body.Data.Order.OrderTime.Time == nil {
		xlog.Warn(ctx, "[MONEY-FLOW-CALC] Transaction time is nil",
			xlog.String("ref_number", notification.Body.Data.Order.RefNumber))
		return nil
	}

	orderTime := *notification.Body.Data.Order.OrderTime.Time
	if orderTime.IsZero() {
		xlog.Warn(ctx, "[MONEY-FLOW-CALC] Invalid transaction time",
			xlog.String("ref_number", notification.Body.Data.Order.RefNumber))
		return nil
	}

	jakartaLocation, _ := time.LoadLocation("Asia/Jakarta")
	creationDate := time.Date(
		orderTime.Year(),
		orderTime.Month(),
		orderTime.Day(),
		0, 0, 0, 0,
		jakartaLocation,
	)

	// Process each transaction
	for _, trx := range notification.Body.Data.Order.Transactions {
		transactionType := string(trx.TransactionType)

		businessRulesData, paymentType, err := mf.CheckEligibleTransaction(ctx, "", transactionType)
		if err != nil {
			return err
		}

		if businessRulesData == nil {
			xlog.Info(ctx, "[MONEY-FLOW-CALC] Skipping ineligible transaction",
				xlog.String("transaction_type", transactionType),
				xlog.String("ref_number", notification.Body.Data.Order.RefNumber))
			continue
		}

		summaryID, err := mf.processSingleTransaction(ctx, trx, paymentType, businessRulesData, creationDate)
		if err != nil {
			xlog.Error(ctx, "[MONEY-FLOW-CALC] Failed to process transaction",
				xlog.String("summary_id", summaryID),
				xlog.String("acuan_transaction_id", trx.Id.String()),
				xlog.Err(err))
			return err
		}
	}

	return nil
}

func (mf *moneyFlowCalc) processSingleTransaction(
	ctx context.Context,
	trx goacuanlib.Transaction,
	paymentType string,
	brd *models.BusinessRuleConfig,
	creationDate time.Time,
) (string, error) {
	transactionType := string(trx.TransactionType)
	amount, _ := trx.Amount.Float64()

	summaryID := uuid.New().String()
	refNumber := constants.MoneyFlowReferencePrefix + creationDate.Format(constants.DateFormatYYYYMMDD) + "-" + summaryID

	var resultSummaryID string
	err := mf.srv.sqlRepo.Atomic(ctx, func(ctx context.Context, r repositories.SQLRepository) error {
		processor := NewTransactionProcessor(r.GetMoneyFlowCalcRepository())

		summaryData := models.CreateMoneyFlowSummary{
			ID:                            summaryID,
			TransactionSourceCreationDate: creationDate,
			TransactionType:               transactionType,
			PaymentType:                   paymentType,
			ReferenceNumber:               refNumber,
			Description:                   brd.RequestToPAPA.Description,
			SourceAccount:                 brd.Source.AccountNumber,
			DestinationAccount:            brd.Destination.AccountNumber,
			TotalTransfer:                 amount,
			PapaTransactionID:             "",
			MoneyFlowStatus:               constants.MoneyFlowStatusPending,
			RequestedDate:                 nil,
			ActualDate:                    nil,
			SourceBankAccountNumber:       brd.Source.BankAccountNumber,
			SourceBankAccountName:         brd.Source.BankAccountName,
			SourceBankName:                brd.Source.BankName,
			DestinationBankAccountNumber:  brd.Destination.BankAccountNumber,
			DestinationBankAccountName:    brd.Destination.BankAccountName,
			DestinationBankName:           brd.Destination.BankName,
		}

		var err error
		resultSummaryID, err = processor.ProcessOrUpdate(ctx, summaryData, trx.Id.String(), trx.Amount)
		return err
	})

	return resultSummaryID, err
}

func (mf *moneyFlowCalc) ProcessTransactionStream(ctx context.Context, event gopaymentlib.Event) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check whether payment type is eligible or not
	businessRulesData, _, err := mf.CheckEligibleTransaction(ctx, event.PaymentType.ConvertSingleAPI().ToString(), "")
	if err != nil {
		return err
	}

	if businessRulesData == nil {
		xlog.Info(ctx, "[MONEY-FLOW-UPDATE] Skipping non-eligible transaction (payment type)",
			xlog.String("status", event.Status.ConvertSingleAPI().ToString()),
			xlog.String("papa_transaction_id", event.ID),
			xlog.String("ref_number", event.ReferenceNumber))
		return nil
	}

	// Get summary ID by PAPA transaction ID
	summaryID, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryIDByPapaTransactionID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to get summary ID by PAPA transaction ID: %w", err)
	}

	if summaryID == "" {
		xlog.Warn(ctx, "[MONEY-FLOW-UPDATE] Summary id not found for transaction",
			xlog.String("papa_transaction_id", event.ID),
			xlog.String("ref_number", event.ReferenceNumber))
		return nil
	}

	status := event.Status.ConvertSingleAPI().ToString()

	var updateReq models.MoneyFlowSummaryUpdate

	if status == "SUCCESSFUL" {
		currentTime := time.Now()

		updateReq = models.MoneyFlowSummaryUpdate{
			MoneyFlowStatus: &status,
			ActualDate:      &currentTime,
		}
	} else {
		updateReq = models.MoneyFlowSummaryUpdate{
			MoneyFlowStatus: &status,
		}
	}

	err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().UpdateSummary(ctx, summaryID, updateReq)
	if err != nil {
		xlog.Error(ctx, "[MONEY-FLOW-UPDATE] Failed to update status",
			xlog.String("summary_id", summaryID),
			xlog.String("papa_transaction_id", event.ID),
			xlog.String("ref_number", event.ReferenceNumber),
			xlog.Err(err))
		return err
	}

	xlog.Info(ctx, "[MONEY-FLOW-UPDATE] Successfully updated money flow status",
		xlog.String("summary_id", summaryID),
		xlog.String("papa_transaction_id", event.ID),
		xlog.String("ref_number", event.ReferenceNumber),
		xlog.String("status", status))

	return nil
}

func (mf *moneyFlowCalc) GetSummariesList(ctx context.Context, opts models.MoneyFlowSummaryFilterOptions) ([]models.MoneyFlowSummaryOut, int, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	mfsRepo := mf.srv.sqlRepo.GetMoneyFlowCalcRepository()

	// Get list
	summaries, err := mfsRepo.GetSummariesList(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get money flow summaries: %w", err)
	}

	// Count total
	total, err := mfsRepo.CountSummaryAll(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count money flow summaries: %w", err)
	}

	return summaries, total, nil
}

func (mf *moneyFlowCalc) GetSummaryDetailBySummaryID(ctx context.Context, summaryID string) (result models.MoneyFlowSummaryDetailBySummaryIDOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	result, err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryDetailBySummaryID(ctx, summaryID)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
		return
	}

	return result, nil
}

func (mf *moneyFlowCalc) GetDetailedTransactionsBySummaryID(ctx context.Context, summaryID string, opts models.DetailedTransactionFilterOptions) ([]models.DetailedTransactionOut, int, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	mfsRepo := mf.srv.sqlRepo.GetMoneyFlowCalcRepository()

	// First, get the summary to check if it has a related failed/rejected summary
	summary, err := mfsRepo.GetSummaryDetailBySummaryID(ctx, summaryID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get summary detail: %w", err)
	}

	// Set the related summary ID in filter options
	opts.SummaryID = summaryID
	opts.RelatedFailedOrRejectedSummaryID = summary.RelatedFailedOrRejectedSummaryID

	// Get detailed transactions (now includes transactions from both summaries)
	transactions, err := mfsRepo.GetDetailedTransactionsBySummaryID(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get detailed transactions: %w", err)
	}

	// Count total (now includes count from both summaries)
	total, err := mfsRepo.CountDetailedTransactions(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count detailed transactions: %w", err)
	}

	return transactions, total, nil
}

func (mf *moneyFlowCalc) UpdateSummary(ctx context.Context, summaryID string, req models.UpdateMoneyFlowSummaryRequest) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Validate request - at least one field must be provided
	if err = req.Validate(); err != nil {
		return err
	}

	// Get current summary details
	currentSummary, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryDetailBySummaryID(ctx, summaryID)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
		return err
	}

	// Validate status transition if status is being changed
	if err = req.ValidateStatusTransition(currentSummary.Status); err != nil {
		xlog.Warn(ctx, "[MONEY-FLOW-UPDATE] Invalid status transition",
			xlog.String("summary_id", summaryID),
			xlog.String("current_status", currentSummary.Status),
			xlog.String("new_status", func() string {
				if req.MoneyFlowStatus != nil {
					return *req.MoneyFlowStatus
				}
				return ""
			}()),
			xlog.Err(err))
		return err
	}

	// NEW VALIDATION: Check for PENDING transactions before current date when transitioning from PENDING to IN_PROGRESS
	if req.MoneyFlowStatus != nil &&
		currentSummary.Status == constants.MoneyFlowStatusPending &&
		*req.MoneyFlowStatus == constants.MoneyFlowStatusInProgress {

		// Check if there are any PENDING transactions before this date with same transaction_type and payment_type
		hasPendingBefore, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().HasPendingTransactionBefore(
			ctx,
			currentSummary.TransactionType,
			currentSummary.PaymentType,
			currentSummary.TransactionSourceCreationDate,
		)
		if err != nil {
			xlog.Error(ctx, "[MONEY-FLOW-UPDATE] Failed to check pending transactions before",
				xlog.String("summary_id", summaryID),
				xlog.String("transaction_type", currentSummary.TransactionType),
				xlog.String("payment_type", currentSummary.PaymentType),
				xlog.Time("transaction_date", currentSummary.TransactionSourceCreationDate),
				xlog.Err(err))
			return fmt.Errorf("failed to check pending transactions: %w", err)
		}

		if hasPendingBefore {
			xlog.Warn(ctx, "[MONEY-FLOW-UPDATE] Blocked status transition due to earlier PENDING transactions",
				xlog.String("summary_id", summaryID),
				xlog.String("transaction_type", currentSummary.TransactionType),
				xlog.String("payment_type", currentSummary.PaymentType),
				xlog.Time("transaction_date", currentSummary.TransactionSourceCreationDate))

			return fmt.Errorf("cannot transition to IN_PROGRESS: there are PENDING transactions with the same payment type (%s) and transaction type (%s) from earlier dates that must be processed first",
				currentSummary.PaymentType,
				currentSummary.TransactionType)
		}
	}

	// Validate IN_PROGRESS requirements
	if err = req.ValidateInProgressRequirements(); err != nil {
		xlog.Warn(ctx, "[MONEY-FLOW-UPDATE] IN_PROGRESS validation failed",
			xlog.String("summary_id", summaryID),
			xlog.Err(err))
		return err
	}

	// Convert request to update model with auto-fill logic
	updateModel, err := req.ToUpdateModelWithAutoFill(currentSummary.RequestedDate)
	if err != nil {
		return err
	}

	// Update summary
	err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().UpdateSummary(ctx, summaryID, *updateModel)
	if err != nil {
		xlog.Error(ctx, "[MONEY-FLOW-UPDATE] Failed to update summary",
			xlog.String("summary_id", summaryID),
			xlog.Err(err))
		return fmt.Errorf("failed to update money flow summary: %w", err)
	}

	xlog.Info(ctx, "[MONEY-FLOW-UPDATE] Successfully updated money flow summary",
		xlog.String("summary_id", summaryID),
		xlog.String("status", func() string {
			if req.MoneyFlowStatus != nil {
				return *req.MoneyFlowStatus
			}
			return currentSummary.Status
		}()))

	return nil
}

// DownloadDetailedTransactionsBySummaryID downloads detailed transactions as CSV
func (mf *moneyFlowCalc) DownloadDetailedTransactionsBySummaryID(ctx context.Context, req models.DownloadDetailedTransactionsRequest) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	mfsRepo := mf.srv.sqlRepo.GetMoneyFlowCalcRepository()

	// Check if summary exists and get related summary info
	summary, err := mfsRepo.GetSummaryDetailBySummaryID(ctx, req.SummaryID)
	if err != nil {
		err = checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
		return err
	}

	// Set related summary ID for download
	req.RelatedFailedOrRejectedSummaryID = summary.RelatedFailedOrRejectedSummaryID

	// Get all detailed transactions (now includes transactions from both summaries)
	transactions, err := mfsRepo.GetAllDetailedTransactionsBySummaryID(
		ctx,
		req.SummaryID,
		req.RelatedFailedOrRejectedSummaryID,
		req.RefNumber,
	)
	if err != nil {
		return fmt.Errorf("failed to get detailed transactions: %w", err)
	}

	// Create CSV writer
	writer := csv.NewWriter(req.Writer)
	defer writer.Flush()

	// Write header
	header := []string{
		"transactionDate",
		"transactionId",
		"reffNumb",
		"typeTransaction",
		"fromAccount",
		"toAccount",
		"amount",
		"description",
		"metadata",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, trx := range transactions {
		record := []string{
			trx.TransactionDate.Format(constants.DateFormatYYYYMMDD),
			trx.TransactionID,
			trx.RefNumber,
			trx.TypeTransaction,
			trx.SourceAccount,
			trx.DestinationAccount,
			trx.Amount.String(),
			trx.Description,
			trx.Metadata,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// loadBusinessRules loads and validates business rules from feature flag
func (mf *moneyFlowCalc) loadBusinessRules(ctx context.Context) (*models.BusinessRulesConfigs, error) {
	variant := mf.srv.flag.GetVariant(mf.srv.conf.FeatureFlagKeyLookup.MoneyFlowCalcBusinessRulesConfig)

	if variant == nil {
		return nil, fmt.Errorf("feature flag variant not found")
	}

	if !variant.Enabled {
		return nil, fmt.Errorf("feature flag variant is disabled")
	}

	if variant.Payload.Value == "" {
		return nil, fmt.Errorf("feature flag variant has empty payload")
	}

	var businessRulesData models.BusinessRulesConfigs
	if err := json.Unmarshal([]byte(variant.Payload.Value), &businessRulesData); err != nil {
		xlog.Error(ctx, "[MONEY-FLOW-CALC] Failed to unmarshal business rules",
			xlog.String("payload", variant.Payload.Value),
			xlog.Err(err))
		return nil, fmt.Errorf("failed to unmarshal business rules: %w", err)
	}

	if len(businessRulesData.PaymentConfigs) == 0 {
		return nil, fmt.Errorf("business rules config is empty")
	}

	return &businessRulesData, nil
}
