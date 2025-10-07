package transformer

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// tupipTransformer is a transformer to acuan transaction for TUPIP transaction type
type tupipTransformer struct {
	baseWalletTransactionTransformer
}

func (t tupipTransformer) Transform(_ context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       parentWalletTransaction.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "TUPIP",
			OrderType:       "TUP",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Metadata:        parentWalletTransaction.Metadata,
			Description:     parentWalletTransaction.Description,
		},
	}, nil
}
