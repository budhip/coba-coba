package transformer

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type rvrslTransformer struct {
	baseWalletTransactionTransformer
}

func (t rvrslTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.TransactionFlow != models.TransactionFlowRefund {
		return nil, common.ErrUnsupportedTransactionFlow
	}

	walletTrxId, ok := parentWalletTransaction.Metadata["walletTransactionId"].(string)
	if !ok || walletTrxId == "" {
		return nil, common.ErrMissingWalletTransactionIdFromMetadata
	}

	limit := t.config.TransactionConfig.ReversalTimeRangeDays
	if limit <= 0 {
		limit = models.DefaultReversalTimeRangeDay
	}

	now, _ := common.NowZeroTime()
	startDate := now.AddDate(0, 0, -limit)

	ts, err := t.transaction.GetList(ctx, models.TransactionFilterOptions{
		Search:    parentWalletTransaction.RefNumber,
		SearchBy:  "refNumber",
		StartDate: &startDate,
		EndDate:   &now,
	})
	if err != nil {
		return nil, err
	}

	if len(ts) == 0 {
		return nil, common.ErrrefNumberNotFound
	}

	isRefundIssuerQRIS := isTransactionContains(ts, "PAYQR")
	if isRefundIssuerQRIS {
		return t.handleRefundIssuerQRIS(ctx, requestRefundIssuerQRIS{
			status:              status,
			userPayload:         parentWalletTransaction,
			walletTransactionId: walletTrxId,
			transactions:        ts,
		})
	}

	for _, transaction := range ts {
		if transaction.TypeTransaction == "RVRSL" {
			return nil, fmt.Errorf("transaction already refunded")
		}

		res = append(res, models.TransactionReq{
			RefNumber:       parentWalletTransaction.RefNumber,
			OrderType:       "RVR",
			TypeTransaction: "RVRSL",
			Description:     transaction.TypeTransaction + " " + transaction.TransactionID,
			Amount:          transaction.Amount,
			FromAccount:     transaction.ToAccount,
			ToAccount:       transaction.FromAccount,
			TransactionID:   uuid.New().String(),
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Status:          string(status),
			OrderTime:       getOrderTime(parentWalletTransaction),
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Metadata:        parentWalletTransaction.Metadata,
		})
	}
	return res, nil
}

type requestRefundIssuerQRIS struct {
	userPayload         models.WalletTransaction
	status              models.TransactionStatus
	walletTransactionId string
	transactions        []models.Transaction
}

func (t rvrslTransformer) handleRefundIssuerQRIS(ctx context.Context, req requestRefundIssuerQRIS) (res []models.TransactionReq, err error) {
	if len(req.userPayload.Amounts) > 0 {
		return nil, fmt.Errorf("unsupported refund qris issuer with child amount")
	}

	wallet, err := t.walletTransaction.GetById(ctx, req.walletTransactionId)
	if err != nil {
		return nil, err
	}

	// get list wallet transaction that have refund process that still pending or success
	wallets, err := t.walletTransaction.List(ctx, models.WalletTrxFilterOptions{
		RefNumber: wallet.RefNumber,
		Limit:     -1,
	})
	if err != nil {
		return nil, err
	}

	inputAmount := req.userPayload.NetAmount.ValueDecimal.Decimal
	originalAmount := getTotalAmount(*wallet, "PAYQR")
	refundedAmount := getTotalAmountFromWallets(wallets, optsCalculateTotalAmount{
		transactionType: "RVRSL", // parent still RVRSL, while child will be RFDQR
		status: []models.WalletTransactionStatus{
			models.WalletTransactionStatusPending,
			models.WalletTransactionStatusSuccess,
		},
	})

	if refundedAmount.Add(inputAmount).GreaterThan(originalAmount) {
		return nil, common.ErrRefundAmountHigherThanOriginalAmount
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       req.userPayload.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(req.userPayload.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(inputAmount),
			Status:          string(req.status),
			TypeTransaction: "RFDQR",
			OrderType:       "RFD",
			OrderTime:       getOrderTime(req.userPayload),
			RefNumber:       req.userPayload.RefNumber,
			Currency:        transformCurrency(req.userPayload.NetAmount.Currency),
			TransactionTime: req.userPayload.TransactionTime,
			Description:     req.userPayload.Description,
			Metadata:        req.userPayload.Metadata,
		},
	}, nil
}
