package transformer

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"context"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

type dsbtfTransformer struct {
	baseWalletTransactionTransformer
}

func (t dsbtfTransformer) Transform(ctx context.Context, amount models.Amount, pwt models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(pwt.Status)
	if err != nil {
		return nil, err
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     pwt.AccountNumber,
			ToAccount:       pwt.DestinationAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(pwt.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "DSBTF",
			OrderType:       "DSB",
			OrderTime:       getOrderTime(pwt),
			RefNumber:       pwt.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: pwt.TransactionTime,
			Description:     pwt.Description,
			Metadata:        pwt.Metadata,
		},
	}, nil
}
