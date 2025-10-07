package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// tuppyTransformer is a transformer to acuan transaction for TUPPY transaction type
type tuppyTransformer struct {
	baseWalletTransactionTransformer
}

func (t tuppyTransformer) Transform(_ context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	metadata := parentWalletTransaction.Metadata
	for _, detail := range parentWalletTransaction.Amounts {
		isAdminFee := slices.Contains([]string{"TUPPU", "ADMME"}, detail.Type)
		if isAdminFee {
			adminFee := detail.Amount.ValueDecimal
			maps.Copy(metadata, map[string]any{
				"accountNumber": parentWalletTransaction.AccountNumber,
				"amount":        amount.ValueDecimal.String(),
				"adminFee":      adminFee.String(),
				"net":           amount.ValueDecimal.Sub(adminFee.Decimal).String(),
			})
			break
		}
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       parentWalletTransaction.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "TUPPY",
			OrderType:       "TUP",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Metadata:        metadata,
			Description:     parentWalletTransaction.Description,
		},
	}, nil
}
