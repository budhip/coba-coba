package services

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/chunkhelper"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
	UpdateActivationStatus(ctx context.Context, summaryID string, isActive bool) error
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

func (mf *moneyFlowCalc) processSingleTransaction(
	ctx context.Context,
	trx goacuanlib.Transaction,
	paymentType string,
	brd *models.BusinessRuleConfig,
) (string, error) {
	transactionType := string(trx.TransactionType)
	amount, _ := trx.Amount.Float64()

	summaryID := uuid.New().String()
	timeNow := time.Now()
	refNumber := constants.MoneyFlowReferencePrefix + timeNow.Format(constants.DateFormatYYYYMMDD) + "-" + summaryID

	var resultSummaryID string
	err := mf.srv.sqlRepo.Atomic(ctx, func(ctx context.Context, r repositories.SQLRepository) error {
		processor := NewTransactionProcessor(r.GetMoneyFlowCalcRepository())

		summaryData := models.CreateMoneyFlowSummary{
			ID:                            summaryID,
			TransactionSourceCreationDate: timeNow,
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

	// Get list (opts.Limit sudah over-fetched dari BuildCursorAndLimit)
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

	// Get summary to check for related failed/rejected summary
	summary, err := mfsRepo.GetSummaryDetailBySummaryID(ctx, summaryID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get summary detail: %w", err)
	}

	opts.SummaryID = summaryID
	opts.RelatedFailedOrRejectedSummaryID = summary.RelatedFailedOrRejectedSummaryID

	// STEP 1: Get transaction IDs with dmfs.id mapping
	idMapping, transactionIDs, err := mfsRepo.GetDetailedTransactionIDsWithMapping(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transaction IDs: %w", err)
	}

	xlog.Info(ctx, "[GET-DETAILED-TRANSACTIONS] Retrieved transaction IDs",
		xlog.String("summary_id", summaryID),
		xlog.Int("ids_count", len(transactionIDs)))

	// STEP 2: Batch fetch from transaction table
	transactions, err := mfsRepo.GetTransactionsByIDs(ctx, transactionIDs, opts.RefNumber)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions by IDs: %w", err)
	}

	// STEP 3: Populate dmfs.id for cursor
	for i := range transactions {
		if dmfsID, exists := idMapping[transactions[i].TransactionID]; exists {
			transactions[i].ID = dmfsID // Set dmfs.id for cursor
		}
	}

	// Use estimated count
	total, err := mfsRepo.EstimateCountDetailedTransactions(ctx, opts)
	if err != nil {
		xlog.Warn(ctx, "[ESTIMATE-COUNT-ERROR] Failed to get estimated count, using 0",
			xlog.String("summary_id", summaryID),
			xlog.Err(err))
		total = 0
	}

	xlog.Info(ctx, "[GET-DETAILED-TRANSACTIONS] Retrieved transactions with estimated count",
		xlog.String("summary_id", summaryID),
		xlog.Int("transactions_count", len(transactions)),
		xlog.Int("estimated_total", total))

	return transactions, total, nil
}

// UpdateSummary validates and updates a money flow summary
func (mf *moneyFlowCalc) UpdateSummary(ctx context.Context, summaryID string, req models.UpdateMoneyFlowSummaryRequest) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Validate request
	validator := NewUpdateValidator(mf.srv.sqlRepo.GetMoneyFlowCalcRepository())
	if err = validator.ValidateRequest(req); err != nil {
		return err
	}

	// Get current summary
	currentSummary, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryDetailBySummaryID(ctx, summaryID)
	if err != nil {
		return checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
	}

	// Validate status transition and requirements
	if err = validator.ValidateTransition(ctx, req, currentSummary); err != nil {
		mf.logValidationError(ctx, summaryID, currentSummary.Status, req, err)
		return err
	}

	// Convert and update
	updateModel, err := req.ToUpdateModelWithAutoFill(currentSummary.RequestedDate)
	if err != nil {
		return err
	}

	if err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().UpdateSummary(ctx, summaryID, *updateModel); err != nil {
		mf.logUpdateError(ctx, summaryID, err)
		return fmt.Errorf("failed to update money flow summary: %w", err)
	}

	mf.logUpdateSuccess(ctx, summaryID, req, currentSummary.Status)
	return nil
}

// logValidationError logs validation errors
func (mf *moneyFlowCalc) logValidationError(
	ctx context.Context,
	summaryID string,
	currentStatus string,
	req models.UpdateMoneyFlowSummaryRequest,
	err error,
) {
	fields := []xlog.Field{
		xlog.String("summary_id", summaryID),
		xlog.String("current_status", currentStatus),
		xlog.Err(err),
	}

	if req.MoneyFlowStatus != nil {
		fields = append(fields, xlog.String("new_status", *req.MoneyFlowStatus))
	}

	xlog.Warn(ctx, constants.LogPrefixMoneyFlowUpdate+" Invalid status transition", fields...)
}

// logUpdateError logs update errors
func (mf *moneyFlowCalc) logUpdateError(ctx context.Context, summaryID string, err error) {
	xlog.Error(ctx, constants.LogPrefixMoneyFlowUpdate+" Failed to update summary",
		xlog.String("summary_id", summaryID),
		xlog.Err(err))
}

// logUpdateSuccess logs successful updates
func (mf *moneyFlowCalc) logUpdateSuccess(
	ctx context.Context,
	summaryID string,
	req models.UpdateMoneyFlowSummaryRequest,
	currentStatus string,
) {
	status := currentStatus
	if req.MoneyFlowStatus != nil {
		status = *req.MoneyFlowStatus
	}

	xlog.Info(ctx, constants.LogPrefixMoneyFlowUpdate+" Successfully updated money flow summary",
		xlog.String("summary_id", summaryID),
		xlog.String("status", status))
}

// UpdateValidator handles validation logic for updates
type UpdateValidator struct {
	repo repositories.MoneyFlowRepository
}

// NewUpdateValidator creates a new update validator
func NewUpdateValidator(repo repositories.MoneyFlowRepository) *UpdateValidator {
	return &UpdateValidator{repo: repo}
}

// ValidateRequest validates the basic update request
func (v *UpdateValidator) ValidateRequest(req models.UpdateMoneyFlowSummaryRequest) error {
	return req.Validate()
}

// ValidateTransition validates status transition and requirements
func (v *UpdateValidator) ValidateTransition(
	ctx context.Context,
	req models.UpdateMoneyFlowSummaryRequest,
	currentSummary models.MoneyFlowSummaryDetailBySummaryIDOut,
) error {
	// Validate status transition
	if err := req.ValidateStatusTransition(currentSummary.Status); err != nil {
		return err
	}

	// Validate IN_PROGRESS requirements
	if err := req.ValidateInProgressRequirements(); err != nil {
		return err
	}

	// commented until uat with user
	// Validate no pending transactions before
	//if err := v.validateNoPendingBefore(ctx, req, currentSummary); err != nil {
	//	return err
	//}

	return nil
}

// validateNoPendingBefore checks for pending transactions before current date
func (v *UpdateValidator) validateNoPendingBefore(
	ctx context.Context,
	req models.UpdateMoneyFlowSummaryRequest,
	currentSummary models.MoneyFlowSummaryDetailBySummaryIDOut,
) error {
	// Only check when transitioning from PENDING to IN_PROGRESS
	if !v.isTransitioningToInProgress(req, currentSummary) {
		return nil
	}

	hasPendingBefore, err := v.repo.HasPendingTransactionBefore(
		ctx,
		currentSummary.TransactionType,
		currentSummary.PaymentType,
		currentSummary.TransactionSourceCreationDate,
	)
	if err != nil {
		return fmt.Errorf("failed to check pending transactions: %w", err)
	}

	if hasPendingBefore {
		return fmt.Errorf(
			constants.ErrMsgPendingTransactionBefore,
			currentSummary.PaymentType,
			currentSummary.TransactionType,
		)
	}

	return nil
}

// isTransitioningToInProgress checks if transitioning to IN_PROGRESS
func (v *UpdateValidator) isTransitioningToInProgress(
	req models.UpdateMoneyFlowSummaryRequest,
	currentSummary models.MoneyFlowSummaryDetailBySummaryIDOut,
) bool {
	return req.MoneyFlowStatus != nil &&
		currentSummary.Status == constants.MoneyFlowStatusPending &&
		*req.MoneyFlowStatus == constants.MoneyFlowStatusInProgress
}

// DownloadDetailedTransactionsBySummaryID downloads detailed transactions as CSV with chunked streaming
func (mf *moneyFlowCalc) DownloadDetailedTransactionsBySummaryID(
	ctx context.Context,
	req models.DownloadDetailedTransactionsRequest,
) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Add timeout protection - 10 minutes for large downloads
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Prepare download request
	downloadReq, err := mf.prepareDownloadRequest(ctx, req)
	if err != nil {
		return err
	}

	xlog.Info(ctx, "[DOWNLOAD-CSV] Starting chunked download",
		xlog.String("summary_id", downloadReq.SummaryID),
		xlog.String("ref_number", downloadReq.RefNumber))

	// Initialize CSV writer
	csvWriter := csv.NewWriter(req.Writer)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(constants.CSVHeaders); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Stream data in chunks
	const chunkSize = 3000 // 3000 rows per chunk
	lastID := ""
	totalWritten := 0
	chunkCount := 0

	progress := &chunkhelper.DownloadProgress{
		SummaryID: downloadReq.SummaryID,
		StartTime: time.Now(),
	}

	for {
		chunkCount++

		xlog.Info(ctx, "[DOWNLOAD-CSV] Fetching chunk",
			xlog.String("summary_id", downloadReq.SummaryID),
			xlog.Int("chunk_number", chunkCount),
			xlog.String("last_id", lastID))

		// Get chunk of transactions
		transactions, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().
			GetDetailedTransactionsChunk(
				ctx,
				downloadReq.SummaryID,
				downloadReq.RelatedFailedOrRejectedSummaryID,
				downloadReq.RefNumber,
				lastID,
				chunkSize,
			)
		if err != nil {
			return fmt.Errorf("failed to get transactions chunk %d: %w", chunkCount, err)
		}

		// Update progress
		progress.TotalChunks = chunkCount
		progress.TotalRows = totalWritten
		progress.LastChunkAt = time.Now()

		// No more data
		if len(transactions) == 0 {
			xlog.Info(ctx, "[DOWNLOAD-CSV] No more data, stopping",
				xlog.Int("chunk_number", chunkCount),
				xlog.Int("total_written", totalWritten))
			break
		}

		// Write chunk to CSV
		for _, trx := range transactions {
			record := mf.transactionToCSVRecord(trx)
			if err := csvWriter.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV row at chunk %d: %w", chunkCount, err)
			}
		}

		totalWritten += len(transactions)
		lastID = transactions[len(transactions)-1].ID // Last dmfs.id for next cursor

		xlog.Info(ctx, "[DOWNLOAD-CSV] Wrote chunk",
			xlog.Int("chunk_number", chunkCount),
			xlog.Int("chunk_size", len(transactions)),
			xlog.Int("total_written", totalWritten),
			xlog.String("last_id", lastID))

		// Flush CSV writer periodically to stream data
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("failed to flush CSV at chunk %d: %w", chunkCount, err)
		}

		// Break if we got less than chunkSize (last chunk)
		if len(transactions) < chunkSize {
			xlog.Info(ctx, "[DOWNLOAD-CSV] Last chunk detected",
				xlog.Int("chunk_number", chunkCount),
				xlog.Int("chunk_size", len(transactions)))
			break
		}

		// Log every 10 chunks
		if chunkCount%10 == 0 {
			progress.LogProgress(ctx, chunkCount, len(transactions))
		}

	}

	xlog.Info(ctx, "[DOWNLOAD-CSV] Download completed successfully",
		xlog.String("summary_id", downloadReq.SummaryID),
		xlog.Int("total_chunks", chunkCount),
		xlog.Int("total_rows", totalWritten))

	return nil
}

// prepareDownloadRequest prepares the download request with related summary info
func (mf *moneyFlowCalc) prepareDownloadRequest(
	ctx context.Context,
	req models.DownloadDetailedTransactionsRequest,
) (models.DownloadDetailedTransactionsRequest, error) {
	summary, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryDetailBySummaryID(ctx, req.SummaryID)
	if err != nil {
		return models.DownloadDetailedTransactionsRequest{}, checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
	}

	req.RelatedFailedOrRejectedSummaryID = summary.RelatedFailedOrRejectedSummaryID
	return req, nil
}

// writeTransactionsToCSV writes transactions to CSV writer
func (mf *moneyFlowCalc) writeTransactionsToCSV(
	writer io.Writer,
	transactions []models.DetailedTransactionCSVOut,
) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(constants.CSVHeaders); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, trx := range transactions {
		record := mf.transactionToCSVRecord(trx)
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// transactionToCSVRecord converts transaction to CSV record
func (mf *moneyFlowCalc) transactionToCSVRecord(trx models.DetailedTransactionCSVOut) []string {
	fmt.Println("TRX.createdAt: ", trx.CreatedAt)
	return []string{
		trx.TransactionDate.Format(constants.DateFormatYYYYMMDD),
		trx.TransactionID,
		trx.RefNumber,
		trx.TypeTransaction,
		trx.SourceAccount,
		trx.DestinationAccount,
		trx.Amount.String(),
		trx.Description,
		trx.Metadata,
		trx.CreatedAt,
	}
}

// ProcessTransactionNotification processes transaction notification with validation
func (mf *moneyFlowCalc) ProcessTransactionNotification(
	ctx context.Context,
	notification goacuanlib.Payload[goacuanlib.DataOrder],
) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Validate notification
	validator := NewNotificationValidator()
	if err = validator.Validate(notification); err != nil {
		return err
	}

	//// Get order time
	//orderTime, err := validator.GetOrderTime(notification)
	//if err != nil {
	//	return err
	//}
	//
	//creationDate := getCreationDate(orderTime)

	// Process each transaction
	//return mf.processTransactions(ctx, notification, creationDate)
	return mf.processTransactions(ctx, notification)
}

// processTransactions processes all transactions in notification
func (mf *moneyFlowCalc) processTransactions(
	ctx context.Context,
	notification goacuanlib.Payload[goacuanlib.DataOrder],
) error {
	for _, trx := range notification.Body.Data.Order.Transactions {
		if err := mf.processEligibleTransaction(ctx, trx, notification.Body.Data.Order.RefNumber); err != nil {
			return err
		}
	}
	return nil
}

// processEligibleTransaction processes a single eligible transaction
func (mf *moneyFlowCalc) processEligibleTransaction(
	ctx context.Context,
	trx goacuanlib.Transaction,
	refNumber string,
) error {
	transactionType := string(trx.TransactionType)

	// Check eligibility
	businessRulesData, paymentType, err := mf.CheckEligibleTransaction(ctx, "", transactionType)
	if err != nil {
		return err
	}

	if businessRulesData == nil {
		xlog.Info(ctx, constants.LogPrefixMoneyFlowCalc+" Skipping ineligible transaction",
			xlog.String("transaction_type", transactionType),
			xlog.String("ref_number", refNumber))
		return nil
	}

	// Process transaction
	_, err = mf.processSingleTransaction(ctx, trx, paymentType, businessRulesData)
	if err != nil {
		xlog.Error(ctx, constants.LogPrefixMoneyFlowCalc+" Failed to process transaction",
			xlog.String("acuan_transaction_id", trx.Id.String()),
			xlog.Err(err))
		return err
	}

	return nil
}

// NotificationValidator validates transaction notifications
type NotificationValidator struct{}

// NewNotificationValidator creates a new notification validator
func NewNotificationValidator() *NotificationValidator {
	return &NotificationValidator{}
}

// Validate validates the notification
func (v *NotificationValidator) Validate(notification goacuanlib.Payload[goacuanlib.DataOrder]) error {
	if len(notification.Body.Data.Order.Transactions) == 0 {
		return nil
	}

	// Validate transaction status
	if notification.Body.Data.Order.Transactions[0].Status != 1 {
		xlog.Info(context.Background(), constants.LogPrefixMoneyFlowCalc+" Skipping non-success transaction",
			xlog.String("status", string(notification.Body.Data.Order.Transactions[0].Status)),
			xlog.String("ref_number", notification.Body.Data.Order.RefNumber))
		return fmt.Errorf("transaction status not success")
	}

	return nil
}

// GetOrderTime gets and validates order time
func (v *NotificationValidator) GetOrderTime(notification goacuanlib.Payload[goacuanlib.DataOrder]) (time.Time, error) {
	if notification.Body.Data.Order.OrderTime.Time == nil {
		return time.Time{}, fmt.Errorf("transaction time is nil")
	}

	orderTime := *notification.Body.Data.Order.OrderTime.Time
	if orderTime.IsZero() {
		return time.Time{}, fmt.Errorf("invalid transaction time")
	}

	return orderTime, nil
}

// getCreationDate gets creation date in Jakarta timezone
func getCreationDate(orderTime time.Time) time.Time {
	jakartaLocation, _ := time.LoadLocation("Asia/Jakarta")
	return time.Date(
		orderTime.Year(),
		orderTime.Month(),
		orderTime.Day(),
		0, 0, 0, 0,
		jakartaLocation,
	)
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

func (mf *moneyFlowCalc) UpdateActivationStatus(ctx context.Context, summaryID string, isActive bool) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check if summary exists and get its details
	summary, err := mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetSummaryDetailBySummaryIDAllStatus(ctx, summaryID)
	if err != nil {
		return checkDatabaseError(err, models.ErrKeySummaryIdnotFound)
	}

	// Validate that only PENDING status can be updated
	if summary.Status != constants.MoneyFlowStatusPending {
		return fmt.Errorf("cannot update activation status: only summaries with PENDING status can be updated, current status is %s", summary.Status)
	}

	// Update status
	err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().UpdateActivationStatus(ctx, summaryID, isActive)
	if err != nil {
		xlog.Error(ctx, constants.LogPrefixMoneyFlowUpdate+" Failed to update status",
			xlog.String("summary_id", summaryID),
			xlog.Bool("is_active", isActive),
			xlog.Err(err))
		return fmt.Errorf("failed to update money flow summary status: %w", err)
	}

	status := "inactive"
	if isActive {
		status = "active"
	}

	xlog.Info(ctx, constants.LogPrefixMoneyFlowUpdate+" Successfully updated money flow summary activation status",
		xlog.String("summary_id", summaryID),
		xlog.String("is_active", status))

	return nil
}
