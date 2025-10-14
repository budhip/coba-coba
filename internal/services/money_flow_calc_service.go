package services

import (
	"context"
	"fmt"
	"time"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type MoneyFlowService interface {
	ProcessTransactionNotification(ctx context.Context, notification goacuanlib.Payload[goacuanlib.DataOrder]) error
	CheckEligibleTransaction(ctx context.Context, breakdownTransactionType string) (*models.BankConfig, error)
}

type moneyFlowCalc service

var _ MoneyFlowService = (*moneyFlowCalc)(nil)

const (
	MoneyFlowStatusPending = "PENDING"
)

func (mf *moneyFlowCalc) CheckEligibleTransaction(ctx context.Context, breakdownTransactionType string) (*models.BankConfig, error) {
	var err error
	var result *models.BankConfig

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	result, err = mf.srv.sqlRepo.GetMoneyFlowCalcRepository().GetBankConfig(ctx, breakdownTransactionType)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank config: %w", err)
	}

	return result, nil
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

		businessRulesData, err := mf.CheckEligibleTransaction(ctx, transactionType)
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

		summaryID, errCreate := mf.processSingleTransaction(ctx, trx, refNumber, businessRulesData, transactionDate)
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

func (mf *moneyFlowCalc) processSingleTransaction(ctx context.Context, trx goacuanlib.Transaction, refNumber string, brd *models.BankConfig, transactionDate time.Time) (string, error) {
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
				PaymentType:                  brd.PaymentType,
				ReferenceNumber:              refNumber,
				Description:                  trx.Description,
				SourceAccount:                brd.SourceAccountNumber,
				DestinationAccount:           brd.DestinationAccountNumber,
				TotalTransfer:                amount,
				PapaTransactionID:            "", // Empty string as per requirement
				MoneyFlowStatus:              MoneyFlowStatusPending,
				RequestedDate:                nil, // NULL as per requirement
				ActualDate:                   nil, // NULL as per requirement
				SourceBankAccountNumber:      brd.SourceBankAccountNumber,
				SourceBankAccountName:        brd.SourceBankAccountName,
				SourceBankName:               brd.SourceBankName,
				DestinationBankAccountNumber: brd.DestinationBankAccountNumber,
				DestinationBankAccountName:   brd.DestinationBankAccountName,
				DestinationBankName:          brd.DestinationBankName,
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
		} else if processedResult != nil && processedResult.MoneyFlowStatus != "SUCCESS" {
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
