package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	gopaymentlib "bitbucket.org/Amartha/go-payment-lib/payment-api/models/event"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
)

type MoneyFlowService interface {
	ProcessTransactionNotification(ctx context.Context, notification goacuanlib.Payload[goacuanlib.DataOrder]) error
	CheckEligibleTransaction(ctx context.Context, paymentType, breakdownTransactionType string) (*models.BusinessRuleConfig, string, error)
	ProcessTransactionStream(ctx context.Context, event gopaymentlib.Event) error
}

type moneyFlowCalc service

var _ MoneyFlowService = (*moneyFlowCalc)(nil)

const (
	MoneyFlowStatusPending = "PENDING"
)

var papaValidStatuses = map[string]bool{
	"SUCCESSFUL": true,
	"REJECTED":   true,
}

func (mf *moneyFlowCalc) CheckEligibleTransaction(ctx context.Context, paymentType, breakdownTransactionType string) (*models.BusinessRuleConfig, string, error) {
	var err error
	var businessConfig *models.BusinessRuleConfig

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	getBusinessRules := mf.srv.flag.GetVariant(mf.srv.conf.FeatureFlagKeyLookup.MoneyFlowCalcBusinessRulesConfig)

	var businessRulesData models.BusinessRulesConfigs
	err = json.Unmarshal([]byte(getBusinessRules.Payload.Value), &businessRulesData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal business rules: %w", err)
	}

	if paymentType == "" {
		businessConfig, paymentType, err = GetBusinessRulesByTransactionType(&businessRulesData, breakdownTransactionType)
		if err != nil {
			return nil, "", err
		}
	} else {
		businessConfig, err = GetBusinessRulesByPaymentType(&businessRulesData, paymentType)
	}

	return businessConfig, paymentType, nil
}

func (mf *moneyFlowCalc) ProcessTransactionNotification(ctx context.Context, notification goacuanlib.Payload[goacuanlib.DataOrder]) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	refNumber := notification.Body.Data.Order.RefNumber

	// Validate notification status
	if notification.Body.Data.Order.Transactions[0].Status != 1 {
		xlog.Info(ctx, "[MONEY-FLOW-CALC] Skipping non-success transaction",
			xlog.String("status", string(notification.Body.Data.Order.Transactions[0].Status)),
			xlog.String("ref_number", refNumber))
		return nil
	}

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
				xlog.String("ref_number", refNumber))
			continue
		}

		// Validate transaction time
		if trx.TransactionTime.Time == nil {
			xlog.Warn(ctx, "[MONEY-FLOW-CALC] Transaction time is nil",
				xlog.String("transaction_id", trx.Id.String()),
				xlog.String("ref_number", refNumber))
			continue
		}

		transactionTime := *trx.TransactionTime.Time
		if transactionTime.IsZero() {
			xlog.Warn(ctx, "[MONEY-FLOW-CALC] Invalid transaction time",
				xlog.String("transaction_id", trx.Id.String()),
				xlog.String("ref_number", refNumber))
			continue
		}

		// Get transaction date (without time component)
		transactionDate := time.Date(
			transactionTime.Year(),
			transactionTime.Month(),
			transactionTime.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		summaryID, errCreate := mf.processSingleTransaction(ctx, trx, refNumber, paymentType, businessRulesData, transactionDate)
		if errCreate != nil {
			return errCreate
		}

		xlog.Error(ctx, "[MONEY-FLOW-CALC] Failed to process transaction",
			xlog.String("summary_id", summaryID),
			xlog.String("acuan_transaction_id", trx.Id.String()),
			xlog.String("ref_number", refNumber),
			xlog.Err(err))
		return err

	}

	return nil
}

func (mf *moneyFlowCalc) processSingleTransaction(ctx context.Context, trx goacuanlib.Transaction, refNumber, paymentType string, brd *models.BusinessRuleConfig, transactionDate time.Time) (string, error) {
	var summaryID string

	transactionType := string(trx.TransactionType)

	amount, _ := trx.Amount.Float64()

	err := mf.srv.sqlRepo.Atomic(ctx, func(ctx context.Context, r repositories.SQLRepository) error {
		mfRepo := r.GetMoneyFlowCalcRepository()

		// Check if transaction already processed
		processedResult, err := mfRepo.GetTransactionProcessed(ctx, transactionType, transactionDate)
		if err != nil {
			return fmt.Errorf("failed to check transaction is processed: %w", err)
		}

		if processedResult == nil {
			summaryID, err := mfRepo.CreateSummary(ctx, models.CreateMoneyFlowSummary{
				TransactionSourceDate:        transactionDate,
				TransactionType:              transactionType,
				PaymentType:                  paymentType,
				ReferenceNumber:              refNumber,
				Description:                  trx.Description,
				SourceAccount:                brd.Source.AccountNumber,
				DestinationAccount:           brd.Destination.AccountNumber,
				TotalTransfer:                amount,
				PapaTransactionID:            "", // Empty string as per requirement
				MoneyFlowStatus:              MoneyFlowStatusPending,
				RequestedDate:                nil, // NULL as per requirement
				ActualDate:                   nil, // NULL as per requirement
				SourceBankAccountNumber:      brd.Source.BankAccountNumber,
				SourceBankAccountName:        brd.Source.BankAccountName,
				SourceBankName:               brd.Source.BankName,
				DestinationBankAccountNumber: brd.Destination.BankAccountNumber,
				DestinationBankAccountName:   brd.Destination.BankAccountName,
				DestinationBankName:          brd.Destination.BankName,
			})
			if err != nil {
				return fmt.Errorf("failed to create summary: %w", err)
			}

			// Create detailed summary
			err = mfRepo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
				SummaryID:          summaryID,
				AcuanTransactionID: trx.Id.String(),
			})
			if err != nil {
				return fmt.Errorf("failed to create detailed summary: %w", err)
			}
		} else if processedResult != nil {
			totalAmount := trx.Amount.Add(processedResult.TotalTransfer)

			updateReq := models.MoneyFlowSummaryUpdate{
				TotalTransfer: &totalAmount,
			}

			err = mfRepo.UpdateSummary(ctx, processedResult.ID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to create summary: %w", err)
			}

			// Create detailed summary
			err = mfRepo.CreateDetailedSummary(ctx, models.CreateDetailedMoneyFlowSummary{
				SummaryID:          processedResult.ID,
				AcuanTransactionID: trx.Id.String(),
			})
			if err != nil {
				return fmt.Errorf("failed to create detailed summary: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return summaryID, nil
}

func (mf *moneyFlowCalc) ProcessTransactionStream(ctx context.Context, event gopaymentlib.Event) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// check whether payment type is eligible or not
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

	updateReq := models.MoneyFlowSummaryUpdate{
		MoneyFlowStatus: &status,
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

// GetBusinessRulesByPaymentType to get config by payment type
func GetBusinessRulesByPaymentType(configs *models.BusinessRulesConfigs, paymentType string) (*models.BusinessRuleConfig, error) {
	config, exists := configs.BusinessRulesConfigs[paymentType]
	if !exists {
		return nil, fmt.Errorf("payment type not found: %s", paymentType)
	}
	return &config, nil
}

// GetBusinessRulesByTransactionType to get config by transaction type
func GetBusinessRulesByTransactionType(configs *models.BusinessRulesConfigs, transactionType string) (*models.BusinessRuleConfig, string, error) {
	paymentType, exists := configs.TransactionToPaymentMap[transactionType]
	if !exists {
		return nil, "", fmt.Errorf("transaction type not found: %s", transactionType)
	}

	paymentConfig, err := GetBusinessRulesByPaymentType(configs, paymentType)
	if err != nil {
		return nil, "", err
	}

	return paymentConfig, paymentType, nil
}
