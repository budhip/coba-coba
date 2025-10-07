package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type inslrTransformer struct {
	baseWalletTransactionTransformer
}

func (t inslrTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.DestinationAccountNumber == "" {
		return nil, common.ErrMissingDestinationAccountNumber
	}

	investedOriginal, err := t.accountingClient.GetInvestedAccountNumber(ctx, parentWalletTransaction.AccountNumber)
	if err != nil {
		return nil, err
	}

	investedRemedial, err := t.accountingClient.GetInvestedAccountNumber(ctx, parentWalletTransaction.DestinationAccountNumber)
	if err != nil {
		return nil, err
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     investedOriginal,
			ToAccount:       investedRemedial,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "INSLR",
			OrderType:       "INS",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
