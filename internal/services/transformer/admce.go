package transformer

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
)

type admceTransformer struct {
	baseWalletTransactionTransformer
}

func (t admceTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	transactionType, err := t.masterDataRepository.GetTransactionType(ctx, parentWalletTransaction.TransactionType)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction type: %w with transaction type code: %s", err, parentWalletTransaction.TransactionType)
	}

	metadata := parentWalletTransaction.Metadata
	maps.Copy(metadata, map[string]any{
		"accountNumber": parentWalletTransaction.AccountNumber,
		"amount":        parentWalletTransaction.NetAmount.ValueDecimal.String(),
		"adminFee":      decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
	})

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "ADMCE",
			OrderType:       "ADM",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     fmt.Sprintf("Admin Fee for %s", transactionType.TransactionTypeName),
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
